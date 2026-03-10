package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// SecurityManager 安全管理器
type SecurityManager struct {
	keyManager *KeyManager
	configDir  string
}

// NewSecurityManager 创建安全管理器
func NewSecurityManager(configDir string) (*SecurityManager, error) {
	keyManager, err := NewKeyManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create key manager: %w", err)
	}

	sm := &SecurityManager{
		keyManager: keyManager,
		configDir:  configDir,
	}

	// 初始化时设置所有敏感文件的权限
	if err := sm.SecureAllFiles(); err != nil {
		return nil, fmt.Errorf("failed to secure files: %w", err)
	}

	return sm, nil
}

// SecureAllFiles 设置所有敏感文件的权限为 0600
func (sm *SecurityManager) SecureAllFiles() error {
	// 需要保护的文件列表
	sensitiveFiles := []string{
		"master.key",
		"login_cache.json",
		"wemedia.db",
		"cache.db",
	}

	for _, filename := range sensitiveFiles {
		filePath := filepath.Join(sm.configDir, filename)

		// 检查文件是否存在
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			continue // 文件不存在，跳过
		}

		// 设置权限为 0600（仅所有者可读写）
		if err := os.Chmod(filePath, 0600); err != nil {
			return fmt.Errorf("failed to set permissions for %s: %w", filename, err)
		}
	}

	return nil
}

// ComputeHMAC 计算文件的 HMAC
func (sm *SecurityManager) ComputeHMAC(data []byte) (string, error) {
	masterKey, err := sm.keyManager.GetMasterKey()
	if err != nil {
		return "", fmt.Errorf("failed to get master key: %w", err)
	}

	h := hmac.New(sha256.New, masterKey)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// VerifyHMAC 验证文件的 HMAC
func (sm *SecurityManager) VerifyHMAC(data []byte, expectedHMAC string) (bool, error) {
	actualHMAC, err := sm.ComputeHMAC(data)
	if err != nil {
		return false, err
	}

	return hmac.Equal([]byte(actualHMAC), []byte(expectedHMAC)), nil
}

// SecureWriteFile 安全地写入文件（加密 + HMAC + 0600权限）
func (sm *SecurityManager) SecureWriteFile(filePath string, data []byte) error {
	masterKey, err := sm.keyManager.GetMasterKey()
	if err != nil {
		return fmt.Errorf("failed to get master key: %w", err)
	}

	// 加密数据
	encryptedData, err := EncryptToZGSWX(data, masterKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	// 计算 HMAC
	hmacValue, err := sm.ComputeHMAC(encryptedData)
	if err != nil {
		return fmt.Errorf("failed to compute HMAC: %w", err)
	}

	// 将 HMAC 附加到加密数据后面（32字节十六进制 = 64字符）
	dataWithHMAC := append(encryptedData, []byte(hmacValue)...)

	// 写入文件，权限 0600
	if err := os.WriteFile(filePath, dataWithHMAC, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// SecureReadFile 安全地读取文件（验证HMAC + 解密）
func (sm *SecurityManager) SecureReadFile(filePath string) ([]byte, error) {
	// 读取文件
	dataWithHMAC, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// HMAC 是最后 64 个字符（32字节的十六进制）
	if len(dataWithHMAC) < 64 {
		return nil, fmt.Errorf("file too short to contain HMAC")
	}

	encryptedData := dataWithHMAC[:len(dataWithHMAC)-64]
	expectedHMAC := string(dataWithHMAC[len(dataWithHMAC)-64:])

	// 验证 HMAC
	valid, err := sm.VerifyHMAC(encryptedData, expectedHMAC)
	if err != nil {
		return nil, fmt.Errorf("failed to verify HMAC: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("HMAC verification failed: file may have been tampered with")
	}

	// 解密数据
	masterKey, err := sm.keyManager.GetMasterKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get master key: %w", err)
	}

	decryptedData, err := DecryptFromZGSWX(encryptedData, masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return decryptedData, nil
}

// GetKeyManager 获取密钥管理器
func (sm *SecurityManager) GetKeyManager() *KeyManager {
	return sm.keyManager
}
