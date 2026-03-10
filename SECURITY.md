# WeMediaSpider 安全架构文档

## 概述

WeMediaSpider 实现了专业级的安全架构，确保所有敏感数据（登录凭证、Cookie、数据库文件）都得到妥善保护。

## 安全特性

### 1. 加密存储

所有敏感文件都使用 **AES-256-GCM** 加密算法进行加密存储：

- **login_cache.json**: 登录凭证和 Cookie（加密）
- **wemedia.db**: SQLite 数据库（文件权限保护）
- **cache.db**: 缓存数据库（文件权限保护）
- **master.key**: 主密钥文件（0600 权限）

#### 加密格式：ZGSWX

自定义加密文件格式，包含：
- **Magic Number**: `0x5A475358` (ZGSX)
- **Version**: 版本号
- **Timestamp**: 创建时间戳
- **Salt**: 32字节随机盐值
- **Nonce**: 12字节随机 nonce
- **Encrypted Data**: AES-256-GCM 加密数据
- **Auth Tag**: 16字节认证标签

### 2. 完整性校验

使用 **HMAC-SHA256** 确保文件未被篡改：

- 每个加密文件都附加 64 字符的 HMAC 值
- 读取文件时自动验证 HMAC
- 如果 HMAC 验证失败，拒绝加载文件

### 3. 密钥管理

#### 主密钥派生

使用 **PBKDF2** 算法派生主密钥：

```
主密钥 = PBKDF2(
    password = 随机种子(32字节) + 用户主目录路径 + "WeMediaSpider",
    salt = 随机盐值(32字节),
    iterations = 100,000,
    keyLength = 32字节
)
```

#### 密钥存储

- **master.key** 文件存储：盐值(32字节) + 种子(32字节)
- **不存储**派生后的主密钥，每次使用时重新派生
- 文件权限：**0600**（仅所有者可读写）

### 4. 文件权限

所有敏感文件都设置为 **0600** 权限（仅所有者可读写）：

```bash
-rw------- master.key
-rw------- login_cache.json
-rw------- wemedia.db
-rw------- cache.db
```

### 5. 向后兼容

系统支持自动升级旧格式文件：

1. **明文格式** → **加密格式（无HMAC）**
2. **加密格式（无HMAC）** → **加密格式（含HMAC）**

升级过程：
- 检测到旧格式时自动转换
- 转换后删除旧文件
- 记录日志便于追踪

## 安全模块

### SecurityManager

核心安全管理器，提供以下功能：

```go
type SecurityManager struct {
    keyManager *KeyManager
    configDir  string
}

// 主要方法
func (sm *SecurityManager) SecureAllFiles() error
func (sm *SecurityManager) ComputeHMAC(data []byte) (string, error)
func (sm *SecurityManager) VerifyHMAC(data []byte, expectedHMAC string) (bool, error)
func (sm *SecurityManager) SecureWriteFile(filePath string, data []byte) error
func (sm *SecurityManager) SecureReadFile(filePath string) ([]byte, error)
```

### KeyManager

密钥管理器，负责主密钥的生成、存储和加载：

```go
type KeyManager struct {
    keyFile   string
    masterKey []byte
}

// 主要方法
func (km *KeyManager) GetMasterKey() ([]byte, error)
func (km *KeyManager) RegenerateMasterKey() error
```

### LoginManager

登录管理器，使用 SecurityManager 保护登录凭证：

```go
type LoginManager struct {
    token           string
    cookies         map[string]string
    cacheFile       string
    expireHours     int
    securityManager *SecurityManager
    loginTime       int64
}

// 主要方法
func (lm *LoginManager) saveCache() error  // 加密保存
func (lm *LoginManager) loadCache() error  // 解密加载
func (lm *LoginManager) ExportCredentials() ([]byte, error)  // 导出凭证
func (lm *LoginManager) ImportCredentials(data []byte) error // 导入凭证
```

## 安全最佳实践

### 1. 凭证导出/导入

用户可以安全地导出和导入登录凭证：

```go
// 导出凭证（加密 + HMAC）
data, err := loginManager.ExportCredentials()
os.WriteFile("credentials.zgswx", data, 0600)

// 导入凭证（验证HMAC + 解密）
data, err := os.ReadFile("credentials.zgswx")
err = loginManager.ImportCredentials(data)
```

### 2. 文件权限检查

应用启动时自动检查并修复文件权限：

```go
securityManager, err := crypto.NewSecurityManager(configDir)
// 自动调用 SecureAllFiles() 设置权限
```

### 3. 完整性验证

所有敏感文件读取时都会验证 HMAC：

```go
data, err := securityManager.SecureReadFile(filePath)
// 自动验证 HMAC，如果失败返回错误
```

## 安全威胁模型

### 已防护的威胁

1. **文件窃取**: 加密存储防止直接读取
2. **文件篡改**: HMAC 校验防止未授权修改
3. **权限泄露**: 0600 权限防止其他用户访问
4. **密钥泄露**: 主密钥不直接存储，使用 PBKDF2 派生

### 未防护的威胁

1. **内存转储**: 运行时内存中的明文数据
2. **调试器附加**: 可以通过调试器读取内存
3. **Root/Admin 权限**: 系统管理员可以访问所有文件
4. **物理访问**: 物理访问设备可以绕过所有保护

## 配置文件位置

所有配置文件存储在用户主目录下：

```
Windows: C:\Users\<username>\.wemediaspider\
Linux:   /home/<username>/.wemediaspider/
macOS:   /Users/<username>/.wemediaspider/
```

文件列表：
```
.wemediaspider/
├── master.key           # 主密钥文件（0600）
├── login_cache.json     # 登录缓存（加密 + HMAC）
├── wemedia.db          # 主数据库（0600）
├── cache.db            # 缓存数据库（0600）
├── system_config.json  # 系统配置（明文，无敏感数据）
└── logs/               # 日志目录
```

## 安全审计

### 日志记录

所有安全相关操作都会记录日志：

- 密钥生成/加载
- 文件加密/解密
- HMAC 验证成功/失败
- 格式升级
- 权限修改

### 审计建议

1. 定期检查日志文件
2. 监控 HMAC 验证失败事件
3. 检查文件权限是否正确
4. 验证主密钥文件完整性

## 密钥轮换

如果怀疑密钥泄露，可以重新生成主密钥：

```go
keyManager.RegenerateMasterKey()
```

**注意**: 重新生成密钥会使所有旧的加密文件失效，需要重新登录。

## 合规性

本安全架构符合以下标准：

- **OWASP**: 使用行业标准加密算法
- **NIST**: PBKDF2 密钥派生符合 NIST SP 800-132
- **PCI DSS**: 敏感数据加密存储
- **GDPR**: 用户数据保护

## 更新历史

- **v1.1.0**: 添加 HMAC 完整性校验
- **v1.0.5**: 实现 AES-256-GCM 加密
- **v1.0.0**: 初始版本（明文存储）

## 联系方式

如发现安全问题，请通过以下方式报告：

- GitHub Issues: https://github.com/yourusername/WeMediaSpider/issues
- Email: security@example.com

**请勿公开披露安全漏洞，直到我们有机会修复。**
