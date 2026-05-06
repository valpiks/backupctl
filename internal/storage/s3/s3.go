package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	s3client "github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/valpiks/backupctl/internal/storage"
)

const multipartChunkSize = 5 * 1024 * 1024

type Storage struct {
	client *s3client.Client
	bucket string
	prefix string
}

type Config struct {
	Bucket         string
	Region         string
	Prefix         string
	Endpoint       string
	ForcePathStyle bool
}

func NewStorage(cfg Config) (*Storage, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}

	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	if accessKey == "" {
		accessKey = "awsadmin"
	}
	if secretKey == "" {
		secretKey = "awsadmin"
	}

	awsCfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	}

	client := s3client.NewFromConfig(awsCfg, func(o *s3client.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = cfg.ForcePathStyle
		}
	})

	return &Storage{
		client: client,
		bucket: cfg.Bucket,
		prefix: strings.TrimSuffix(cfg.Prefix, "/"),
	}, nil
}

func (s *Storage) Save(ctx context.Context, name string, data io.Reader) error {
	key := s.key(name)

	firstPart := make([]byte, multipartChunkSize)
	n, readErr := io.ReadFull(data, firstPart)
	switch {
	case readErr == io.EOF && n == 0:
		_, err := s.client.PutObject(ctx, &s3client.PutObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(nil),
		})
		if err != nil {
			return fmt.Errorf("upload %s: %w", name, err)
		}
		return nil
	case readErr == io.ErrUnexpectedEOF:
		_, err := s.client.PutObject(ctx, &s3client.PutObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(firstPart[:n]),
		})
		if err != nil {
			return fmt.Errorf("upload %s: %w", name, err)
		}
		return nil
	case readErr != nil:
		return fmt.Errorf("read upload body %s: %w", name, readErr)
	}

	createOutput, err := s.client.CreateMultipartUpload(ctx, &s3client.CreateMultipartUploadInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("create multipart upload %s: %w", name, err)
	}

	uploadID := aws.ToString(createOutput.UploadId)
	aborted := true
	defer func() {
		if aborted {
			_, _ = s.client.AbortMultipartUpload(ctx, &s3client.AbortMultipartUploadInput{
				Bucket:   aws.String(s.bucket),
				Key:      aws.String(key),
				UploadId: aws.String(uploadID),
			})
		}
	}()

	parts := make([]s3types.CompletedPart, 0, 8)
	partNumber := int32(1)

	uploadPart := func(body []byte) error {
		partOutput, err := s.client.UploadPart(ctx, &s3client.UploadPartInput{
			Bucket:        aws.String(s.bucket),
			Key:           aws.String(key),
			UploadId:      aws.String(uploadID),
			PartNumber:    aws.Int32(partNumber),
			Body:          bytes.NewReader(body),
			ContentLength: aws.Int64(int64(len(body))),
		})
		if err != nil {
			return fmt.Errorf("upload part %d for %s: %w", partNumber, name, err)
		}

		parts = append(parts, s3types.CompletedPart{
			ETag:       partOutput.ETag,
			PartNumber: aws.Int32(partNumber),
		})
		partNumber++
		return nil
	}

	if err := uploadPart(firstPart[:n]); err != nil {
		return err
	}

	for {
		chunk := make([]byte, multipartChunkSize)
		n, readErr := io.ReadFull(data, chunk)
		switch {
		case readErr == io.EOF:
			_, err = s.client.CompleteMultipartUpload(ctx, &s3client.CompleteMultipartUploadInput{
				Bucket:   aws.String(s.bucket),
				Key:      aws.String(key),
				UploadId: aws.String(uploadID),
				MultipartUpload: &s3types.CompletedMultipartUpload{
					Parts: parts,
				},
			})
			if err != nil {
				return fmt.Errorf("complete multipart upload %s: %w", name, err)
			}
			aborted = false
			return nil
		case readErr == io.ErrUnexpectedEOF:
			if n > 0 {
				if err := uploadPart(chunk[:n]); err != nil {
					return err
				}
			}
			_, err = s.client.CompleteMultipartUpload(ctx, &s3client.CompleteMultipartUploadInput{
				Bucket:   aws.String(s.bucket),
				Key:      aws.String(key),
				UploadId: aws.String(uploadID),
				MultipartUpload: &s3types.CompletedMultipartUpload{
					Parts: parts,
				},
			})
			if err != nil {
				return fmt.Errorf("complete multipart upload %s: %w", name, err)
			}
			aborted = false
			return nil
		case readErr != nil:
			return fmt.Errorf("read upload body %s: %w", name, readErr)
		}

		if err := uploadPart(chunk[:n]); err != nil {
			return err
		}
	}
}

func (s *Storage) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	key := s.key(name)

	output, err := s.client.GetObject(ctx, &s3client.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", name, err)
	}

	return output.Body, nil
}

func (s *Storage) List(ctx context.Context) ([]storage.BackupFile, error) {
	var files []storage.BackupFile

	paginator := s3client.NewListObjectsV2Paginator(s.client, &s3client.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(s.listPrefix()),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list objects: %w", err)
		}

		for _, obj := range page.Contents {
			name := strings.TrimPrefix(aws.ToString(obj.Key), s.listPrefix())

			files = append(files, storage.BackupFile{
				Name: name,
				Size: aws.ToInt64(obj.Size),
			})
		}
	}

	return files, nil
}

func (s *Storage) Delete(ctx context.Context, name string) error {
	key := s.key(name)

	_, err := s.client.DeleteObject(ctx, &s3client.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete %s: %w", name, err)
	}

	return nil
}

func (s *Storage) key(name string) string {
	if s.prefix == "" {
		return name
	}

	return s.prefix + "/" + name
}

func (s *Storage) listPrefix() string {
	if s.prefix == "" {
		return ""
	}

	return s.prefix + "/"
}
