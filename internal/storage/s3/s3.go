package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	s3client "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/valpiks/backupctl/internal/storage"
)

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
		prefix: cfg.Prefix,
	}, nil
}

func (s *Storage) Save(ctx context.Context, name string, data io.Reader) error {
	key := s.key(name)

	body, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("read data: %w", err)
	}

	_, err = s.client.PutObject(ctx, &s3client.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	})
	if err != nil {
		return fmt.Errorf("upload %s: %w", name, err)
	}

	return nil
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
		Prefix: aws.String(s.prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list objects: %w", err)
		}

		for _, obj := range page.Contents {
			name := *obj.Key
			if s.prefix != "" {
				name = name[len(s.prefix):]
			}

			files = append(files, storage.BackupFile{
				Name: name,
				Size: *obj.Size,
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
	return s.prefix + name
}
