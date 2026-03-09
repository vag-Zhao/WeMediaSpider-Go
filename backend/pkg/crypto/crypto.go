package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

const (
	saltSize   = 32
	keySize    = 32
	iterations = 100000

	// ZGSWX 文件格式常量
	MagicNumber    = 0x5A475358 // "ZGSX"
	CurrentVersion = 0x0001
	HeaderSize     = 80
	ZGSWXSaltSize  = 32
	ZGSWXNonceSize = 12
	ZGSWXTagSize   = 16
)

// Encrypt 使用密码加密数据
func Encrypt(plaintext []byte, password string) (string, error) {
	// 生成随机盐
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	// 使用 PBKDF2 派生密钥
	key := pbkdf2.Key([]byte(password), salt, iterations, keySize, sha256.New)

	// 创建 AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 使用 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 加密数据
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// 组合: salt + ciphertext
	result := append(salt, ciphertext...)

	// Base64 编码
	return base64.StdEncoding.EncodeToString(result), nil
}

// Decrypt 使用密码解密数据
func Decrypt(encryptedData string, password string) ([]byte, error) {
	// Base64 解码
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}

	// 检查数据长度
	if len(data) < saltSize {
		return nil, errors.New("invalid encrypted data")
	}

	// 分离 salt 和 ciphertext
	salt := data[:saltSize]
	ciphertext := data[saltSize:]

	// 使用 PBKDF2 派生密钥
	key := pbkdf2.Key([]byte(password), salt, iterations, keySize, sha256.New)

	// 创建 AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 使用 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 检查 nonce 大小
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("invalid ciphertext")
	}

	// 分离 nonce 和实际密文
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// 解密数据
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed: invalid password or corrupted data")
	}

	return plaintext, nil
}

// ZGSWXHeader .zgswx 文件头结构
type ZGSWXHeader struct {
	Magic      uint32
	Version    uint16
	Reserved   uint16
	Timestamp  int64
	DataLength uint32
	Salt       [ZGSWXSaltSize]byte
	Nonce      [ZGSWXNonceSize]byte
}

// EncryptToZGSWX 加密数据为 .zgswx 格式
func EncryptToZGSWX(plaintext []byte, masterKey []byte) ([]byte, error) {
	if len(masterKey) != keySize {
		return nil, fmt.Errorf("invalid master key size: expected %d, got %d", keySize, len(masterKey))
	}

	// 生成随机盐值
	var salt [ZGSWXSaltSize]byte
	if _, err := io.ReadFull(rand.Reader, salt[:]); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// 使用 PBKDF2 派生加密密钥
	encKey := pbkdf2.Key(masterKey, salt[:], iterations, keySize, sha256.New)

	// 创建 AES cipher
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// 使用 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// 生成随机 nonce
	var nonce [ZGSWXNonceSize]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 加密数据
	ciphertext := gcm.Seal(nil, nonce[:], plaintext, nil)

	// 构建文件头
	header := ZGSWXHeader{
		Magic:      MagicNumber,
		Version:    CurrentVersion,
		Reserved:   0,
		Timestamp:  time.Now().Unix(),
		DataLength: uint32(len(ciphertext)),
		Salt:       salt,
		Nonce:      nonce,
	}

	// 序列化文件头
	buf := make([]byte, HeaderSize)
	binary.BigEndian.PutUint32(buf[0:4], header.Magic)
	binary.BigEndian.PutUint16(buf[4:6], header.Version)
	binary.BigEndian.PutUint16(buf[6:8], header.Reserved)
	binary.BigEndian.PutUint64(buf[8:16], uint64(header.Timestamp))
	binary.BigEndian.PutUint32(buf[16:20], header.DataLength)
	copy(buf[20:52], header.Salt[:])
	copy(buf[52:64], header.Nonce[:])

	// 组合：头部 + 密文
	result := append(buf, ciphertext...)

	return result, nil
}

// DecryptFromZGSWX 从 .zgswx 格式解密数据
func DecryptFromZGSWX(data []byte, masterKey []byte) ([]byte, error) {
	if len(masterKey) != keySize {
		return nil, fmt.Errorf("invalid master key size: expected %d, got %d", keySize, len(masterKey))
	}

	// 验证文件格式
	if err := ValidateZGSWXFormat(data); err != nil {
		return nil, err
	}

	// 解析文件头
	magic := binary.BigEndian.Uint32(data[0:4])
	version := binary.BigEndian.Uint16(data[4:6])
	timestamp := int64(binary.BigEndian.Uint64(data[8:16]))
	dataLength := binary.BigEndian.Uint32(data[16:20])

	var salt [ZGSWXSaltSize]byte
	var nonce [ZGSWXNonceSize]byte
	copy(salt[:], data[20:52])
	copy(nonce[:], data[52:64])

	// 验证版本
	if version != CurrentVersion {
		return nil, fmt.Errorf("unsupported version: %d", version)
	}

	// 验证魔数
	if magic != MagicNumber {
		return nil, fmt.Errorf("invalid magic number: 0x%X", magic)
	}

	// 提取密文
	ciphertext := data[HeaderSize:]
	if uint32(len(ciphertext)) != dataLength {
		return nil, fmt.Errorf("data length mismatch: expected %d, got %d", dataLength, len(ciphertext))
	}

	// 使用 PBKDF2 派生解密密钥
	encKey := pbkdf2.Key(masterKey, salt[:], iterations, keySize, sha256.New)

	// 创建 AES cipher
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// 使用 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// 解密数据
	plaintext, err := gcm.Open(nil, nonce[:], ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	// 验证时间戳（可选，用于调试）
	_ = timestamp

	return plaintext, nil
}

// ValidateZGSWXFormat 验证 .zgswx 文件格式
func ValidateZGSWXFormat(data []byte) error {
	// 检查最小长度
	if len(data) < HeaderSize {
		return fmt.Errorf("file too small: expected at least %d bytes, got %d", HeaderSize, len(data))
	}

	// 验证魔数
	magic := binary.BigEndian.Uint32(data[0:4])
	if magic != MagicNumber {
		return fmt.Errorf("invalid magic number: expected 0x%X, got 0x%X", MagicNumber, magic)
	}

	// 验证版本
	version := binary.BigEndian.Uint16(data[4:6])
	if version != CurrentVersion {
		return fmt.Errorf("unsupported version: %d", version)
	}

	return nil
}

// WriteZGSWXFile 写入 .zgswx 文件
func WriteZGSWXFile(filepath string, data []byte) error {
	// 验证数据格式
	if err := ValidateZGSWXFormat(data); err != nil {
		return fmt.Errorf("invalid ZGSWX data: %w", err)
	}

	// 写入文件，权限 0600
	if err := os.WriteFile(filepath, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ReadZGSWXFile 读取 .zgswx 文件
func ReadZGSWXFile(filepath string) ([]byte, error) {
	// 读取文件
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 验证文件格式
	if err := ValidateZGSWXFormat(data); err != nil {
		return nil, fmt.Errorf("invalid ZGSWX file: %w", err)
	}

	return data, nil
}
