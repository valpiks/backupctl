package encryption

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"io"
	"strings"
	"testing"
)

func TestAESGCMEncryptDecryptRoundTrip(t *testing.T) {
	encryptor := NewAESGCMEncryptor()

	encrypted, err := encryptor.Encrypt(strings.NewReader("hello backup"), "password")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	defer encrypted.Close()

	decrypted, err := encryptor.Decrypt(encrypted, "password")
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	defer decrypted.Close()

	data, err := io.ReadAll(decrypted)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(data) != "hello backup" {
		t.Fatalf("decrypted = %q", string(data))
	}
}

func TestAESGCMEncryptUsesStreamingMagic(t *testing.T) {
	encryptor := NewAESGCMEncryptor()

	encrypted, err := encryptor.Encrypt(strings.NewReader("hello backup"), "password")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	defer encrypted.Close()

	data, err := io.ReadAll(encrypted)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if !bytes.HasPrefix(data, []byte(Magic)) {
		t.Fatalf("encrypted data magic = %q, want %q", string(data[:len(Magic)]), Magic)
	}
}

func TestAESGCMEncryptDecryptLargeRoundTrip(t *testing.T) {
	encryptor := NewAESGCMEncryptor()

	input := strings.Repeat("large-backup-payload-", ChunkSize/4)

	encrypted, err := encryptor.Encrypt(strings.NewReader(input), "password")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	defer encrypted.Close()

	decrypted, err := encryptor.Decrypt(encrypted, "password")
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	defer decrypted.Close()

	data, err := io.ReadAll(decrypted)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(data) != input {
		t.Fatal("decrypted large payload does not match input")
	}
}

func TestAESGCMDecryptWrongPassword(t *testing.T) {
	encryptor := NewAESGCMEncryptor()

	encrypted, err := encryptor.Encrypt(strings.NewReader("hello backup"), "password")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	defer encrypted.Close()

	decrypted, err := encryptor.Decrypt(encrypted, "wrong-password")
	if err != nil {
		t.Fatalf("Decrypt() setup error = %v", err)
	}
	defer decrypted.Close()

	_, err = io.ReadAll(decrypted)
	if err == nil {
		t.Fatal("ReadAll() error = nil")
	}
}

func TestAESGCMEncryptUsesRandomSaltAndNonce(t *testing.T) {
	encryptor := NewAESGCMEncryptor()

	a, err := encryptor.Encrypt(strings.NewReader("same"), "password")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	defer a.Close()

	b, err := encryptor.Encrypt(strings.NewReader("same"), "password")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	defer b.Close()

	aData, err := io.ReadAll(a)
	if err != nil {
		t.Fatalf("ReadAll(a) error = %v", err)
	}

	bData, err := io.ReadAll(b)
	if err != nil {
		t.Fatalf("ReadAll(b) error = %v", err)
	}

	if string(aData) == string(bData) {
		t.Fatal("encrypted outputs are equal, want random salt/nonce")
	}
}

func TestAESGCMDecryptLegacyFormat(t *testing.T) {
	legacyData, err := encryptLegacyForTest("hello legacy", "password")
	if err != nil {
		t.Fatalf("encryptLegacyForTest() error = %v", err)
	}

	encryptor := NewAESGCMEncryptor()

	decrypted, err := encryptor.Decrypt(bytes.NewReader(legacyData), "password")
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	defer decrypted.Close()

	data, err := io.ReadAll(decrypted)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(data) != "hello legacy" {
		t.Fatalf("decrypted = %q", string(data))
	}
}

func encryptLegacyForTest(input string, password string) ([]byte, error) {
	salt := bytes.Repeat([]byte{1}, SaltSize)
	nonce := bytes.Repeat([]byte{2}, NonceSize)
	key := deriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(input), nil)

	var out bytes.Buffer
	out.WriteString(MagicLegacy)
	out.Write(salt)
	out.Write(nonce)

	if err := binary.Write(&out, binary.BigEndian, uint64(len(ciphertext))); err != nil {
		return nil, err
	}

	out.Write(ciphertext)

	return out.Bytes(), nil
}
