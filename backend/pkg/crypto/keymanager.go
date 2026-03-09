package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/pbkdf2"
)

const (
	masterKeySize       = 32       // 256 bits
	masterKeySaltSize   = 32       // 256 bits
	masterKeyIterations = 100000   // PBKDF2 iterations
)

// KeyManager 密钥管理器
type KeyManager struct {
	keyFile   string
	masterKey []byte
}

// NewKeyManager 创建密钥管理器
func NewKeyManager() (*KeyManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".wemediaspider")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	keyFile := filepath.Join(cacheDir, "master.key")

	km := &KeyManager{
		keyFile: keyFile,
	}

	// 尝试加载现有密钥，如果不存在则生成新密钥
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		if err := km.generateAndSaveMasterKey(); err != nil {
			return nil, err
		}
	} else {
		if err := km.loadMasterKey(); err != nil {
			return nil, err
		}
	}

	return km, nil
}

// GetMasterKey 获取主密钥
func (km *KeyManager) GetMasterKey() ([]byte, error) {
	if km.masterKey == nil {
		return nil, fmt.Errorf("master key not initialized")
	}
	return km.masterKey, nil
}

// generateAndSaveMasterKey 生成并保存主密钥
func (km *KeyManager) generateAndSaveMasterKey() error {
	// 生成随机种子
	seed := make([]byte, masterKeySize)
	if _, err := rand.Read(seed); err != nil {
		return fmt.Errorf("failed to generate random seed: %w", err)
	}

	// 生成随机盐值
	salt := make([]byte, masterKeySaltSize)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// 获取用户主目录路径作为额外熵源
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// 组合：随机种子 + 用户主目录路径 + 应用标识
	combined := append(seed, []byte(homeDir)...)
	combined = append(combined, []byte("WeMediaSpider")...)

	// 使用 PBKDF2 派生最终密钥
	masterKey := pbkdf2.Key(combined, salt, masterKeyIterations, masterKeySize, sha256.New)

	// 保存密钥文件：盐值 + 种子（不保存派生后的密钥）
	keyData := append(salt, seed...)
	if err := os.WriteFile(km.keyFile, keyData, 0600); err != nil {
		return fmt.Errorf("failed to save master key: %w", err)
	}

	km.masterKey = masterKey
	return nil
}

// loadMasterKey 从文件加载主密钥
func (km *KeyManager) loadMasterKey() error {
	// 读取密钥文件
	keyData, err := os.ReadFile(km.keyFile)
	if err != nil {
		return fmt.Errorf("failed to read master key file: %w", err)
	}

	// 验证文件大小
	expectedSize := masterKeySaltSize + masterKeySize
	if len(keyData) != expectedSize {
		return fmt.Errorf("invalid master key file size: expected %d, got %d", expectedSize, len(keyData))
	}

	// 分离盐值和种子
	salt := keyData[:masterKeySaltSize]
	seed := keyData[masterKeySaltSize:]

	// 获取用户主目录路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// 重新组合并派生密钥
	combined := append(seed, []byte(homeDir)...)
	combined = append(combined, []byte("WeMediaSpider")...)

	masterKey := pbkdf2.Key(combined, salt, masterKeyIterations, masterKeySize, sha256.New)

	km.masterKey = masterKey
	return nil
}

// RegenerateMasterKey 重新生成主密钥（会使旧凭证文件失效）
func (km *KeyManager) RegenerateMasterKey() error {
	// 删除旧密钥文件
	if err := os.Remove(km.keyFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old master key: %w", err)
	}

	// 生成新密钥
	return km.generateAndSaveMasterKey()
}
