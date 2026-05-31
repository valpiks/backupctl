package encryption

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"golang.org/x/crypto/argon2"
)

const (
	MagicLegacy    = "BCTLENC1"
	Magic          = "BCTLENC2"
	SaltSize       = 16
	NonceSize      = 12
	KeySize        = 32
	ArgonTime      = 1
	ArgonMemory    = 64 * 1024
	ArgonThreads   = 4
	ChunkSize      = 64 * 1024
	chunkLengthLen = 4
)

type AESGCMEncryptor struct{}

func NewAESGCMEncryptor() *AESGCMEncryptor {
	return &AESGCMEncryptor{}
}

func (e *AESGCMEncryptor) Encrypt(reader io.Reader, password string) (io.ReadCloser, error) {
	if password == "" {
		return nil, fmt.Errorf("encryption password is required")
	}

	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	baseNonce := make([]byte, NonceSize)
	if _, err := rand.Read(baseNonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	key := deriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	pipeReader, pipeWriter := io.Pipe()

	go func() {
		err := encryptStream(pipeWriter, reader, gcm, salt, baseNonce)
		_ = pipeWriter.CloseWithError(err)
	}()

	return pipeReader, nil
}

func (e *AESGCMEncryptor) Decrypt(reader io.Reader, password string) (io.ReadCloser, error) {
	if password == "" {
		return nil, fmt.Errorf("encryption password is required")
	}

	magic := make([]byte, len(Magic))
	if _, err := io.ReadFull(reader, magic); err != nil {
		return nil, fmt.Errorf("read encrypted data header: %w", err)
	}

	switch string(magic) {
	case Magic:
		return e.decryptStream(reader, password)
	case MagicLegacy:
		legacyReader := io.MultiReader(bytes.NewReader(magic), reader)
		return e.decryptLegacy(legacyReader, password)
	default:
		return nil, fmt.Errorf("invalid encrypted data header")
	}
}

func encryptStream(writer io.Writer, reader io.Reader, gcm cipher.AEAD, salt []byte, baseNonce []byte) error {
	if _, err := writer.Write([]byte(Magic)); err != nil {
		return fmt.Errorf("write encrypted data header: %w", err)
	}

	if _, err := writer.Write(salt); err != nil {
		return fmt.Errorf("write salt: %w", err)
	}

	if _, err := writer.Write(baseNonce); err != nil {
		return fmt.Errorf("write nonce: %w", err)
	}

	buffer := make([]byte, ChunkSize)
	var counter uint64

	for {
		n, readErr := reader.Read(buffer)
		if n > 0 {
			if counter == math.MaxUint64 {
				return fmt.Errorf("encrypted data has too many chunks")
			}

			nonce := chunkNonce(baseNonce, counter)
			ciphertext := gcm.Seal(nil, nonce, buffer[:n], nil)

			if len(ciphertext) > math.MaxUint32 {
				return fmt.Errorf("encrypted chunk is too large")
			}

			if err := binary.Write(writer, binary.BigEndian, uint32(len(ciphertext))); err != nil {
				return fmt.Errorf("write encrypted chunk length: %w", err)
			}

			if _, err := writer.Write(ciphertext); err != nil {
				return fmt.Errorf("write encrypted chunk: %w", err)
			}

			counter++
		}

		if readErr == io.EOF {
			if err := binary.Write(writer, binary.BigEndian, uint32(0)); err != nil {
				return fmt.Errorf("write encrypted stream terminator: %w", err)
			}
			return nil
		}

		if readErr != nil {
			return fmt.Errorf("read plaintext: %w", readErr)
		}
	}
}

func (e *AESGCMEncryptor) decryptStream(reader io.Reader, password string) (io.ReadCloser, error) {
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(reader, salt); err != nil {
		return nil, fmt.Errorf("read salt: %w", err)
	}

	baseNonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(reader, baseNonce); err != nil {
		return nil, fmt.Errorf("read nonce: %w", err)
	}

	key := deriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	pipeReader, pipeWriter := io.Pipe()

	go func() {
		err := decryptStream(pipeWriter, reader, gcm, baseNonce)
		_ = pipeWriter.CloseWithError(err)
	}()

	return pipeReader, nil
}

func decryptStream(writer io.Writer, reader io.Reader, gcm cipher.AEAD, baseNonce []byte) error {
	lengthBuf := make([]byte, chunkLengthLen)
	var counter uint64

	for {
		if _, err := io.ReadFull(reader, lengthBuf); err != nil {
			return fmt.Errorf("read encrypted chunk length: %w", err)
		}

		chunkLen := binary.BigEndian.Uint32(lengthBuf)
		if chunkLen == 0 {
			return nil
		}

		ciphertext := make([]byte, chunkLen)
		if _, err := io.ReadFull(reader, ciphertext); err != nil {
			return fmt.Errorf("read encrypted chunk: %w", err)
		}

		nonce := chunkNonce(baseNonce, counter)
		plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return fmt.Errorf("decrypt chunk: %w", err)
		}

		if _, err := writer.Write(plaintext); err != nil {
			return fmt.Errorf("write plaintext chunk: %w", err)
		}

		counter++
	}
}

func (e *AESGCMEncryptor) decryptLegacy(reader io.Reader, password string) (io.ReadCloser, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read ciphertext: %w", err)
	}

	headerSize := len(MagicLegacy) + SaltSize + NonceSize + 8
	if len(data) < headerSize {
		return nil, fmt.Errorf("encrypted data is too short")
	}

	if string(data[:len(MagicLegacy)]) != MagicLegacy {
		return nil, fmt.Errorf("invalid encrypted data header")
	}

	offset := len(MagicLegacy)

	salt := data[offset : offset+SaltSize]
	offset += SaltSize

	nonce := data[offset : offset+NonceSize]
	offset += NonceSize

	ciphertextLen := binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8

	if uint64(len(data[offset:])) != ciphertextLen {
		return nil, fmt.Errorf("encrypted data length mismatch")
	}

	ciphertext := data[offset:]

	key := deriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt data: %w", err)
	}

	return io.NopCloser(bytes.NewReader(plaintext)), nil
}

func deriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, ArgonTime, ArgonMemory, ArgonThreads, KeySize)
}

func chunkNonce(baseNonce []byte, counter uint64) []byte {
	nonce := make([]byte, NonceSize)
	copy(nonce, baseNonce)
	binary.BigEndian.PutUint64(nonce[NonceSize-8:], counter)
	return nonce
}
