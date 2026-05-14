package compression

import (
	"compress/gzip"
	"io"
)

type GzipCompressor struct{}

func NewGzipCompressor() *GzipCompressor {
	return &GzipCompressor{}
}

func (c *GzipCompressor) Compress(input io.Reader) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		gz := gzip.NewWriter(pw)

		_, copyErr := io.Copy(gz, input)
		closeErr := gz.Close()

		if copyErr != nil {
			_ = pw.CloseWithError(copyErr)
			return
		}

		if closeErr != nil {
			_ = pw.CloseWithError(closeErr)
			return
		}

		_ = pw.Close()
	}()

	return pr
}

func (c *GzipCompressor) Decompress(input io.Reader) (io.ReadCloser, error) {
	return gzip.NewReader(input)
}
