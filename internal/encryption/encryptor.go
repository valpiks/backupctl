package encryption

import "io"

type Encryptor interface {
	Encrypt(reader io.Reader, password string) (io.ReadCloser, error)
	Decrypt(reader io.Reader, password string) (io.ReadCloser, error)
}
