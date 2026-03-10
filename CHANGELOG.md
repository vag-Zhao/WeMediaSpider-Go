# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2026-03-10

### 🚀 Major Changes
- **数据库迁移**: 从 JSON 文件存储迁移到 SQLite 数据库
  - 使用 GORM ORM 框架进行数据访问
  - 实现 Repository 模式，提供清晰的数据访问层
  - 支持高效的查询、分页、搜索功能
  - 自动数据迁移工具（`backend/cmd/migrate/main.go`）

### 🔐 Security Enhancements
- **专业级安全架构**
  - 登录凭证使用 AES-256-GCM 加密存储
  - 添加 HMAC-SHA256 完整性校验，防止文件篡改
  - 所有敏感文件设置 0600 权限（仅所有者可读写）
  - 实现 SecurityManager 统一管理加密和完整性验证
  - 支持旧格式自动升级（明文 → 加密 → 加密+HMAC）
  - 详细安全文档（SECURITY.md）

### ✨ Features
- **数据按公众号分组**
  - 导入历史数据界面按公众号分组显示
  - 显示每个公众号的文章数量和时间跨度
  - 支持展开查看该公众号的所有文章

- **UI/UX 优化**
  - 优化卡片布局，缩小 Logo 尺寸，降低卡片高度
  - 文章列表支持容器内滚动（最大高度 400px）
  - 修复滚动穿透问题，防止页面滚动
  - 展开时固定卡片标题位置（sticky positioning）
  - 时间和标签垂直居中对齐
  - 自定义滚动条样式（6px 宽度，圆角设计）
  - 移除冗余的"公众号列表"和"文章列表"标题

### 🗄️ Database
- **新增数据库模块**
  - `backend/internal/database/db.go` - 数据库连接管理
  - `backend/internal/database/models/` - GORM 数据模型
    - `account.go` - 公众号模型
    - `article.go` - 文章模型
    - `app_stats.go` - 应用统计模型
  - `backend/internal/repository/` - 数据访问层
    - `article_repo.go` - 文章仓储
    - `account_repo.go` - 公众号仓储
    - `stats_repo.go` - 统计仓储

- **数据库特性**
  - WAL 模式提升并发性能
  - 连接池管理（最大 25 连接）
  - 批量操作优化（每批 500 条）
  - 索引优化（article_id, account_id, publish_timestamp）
  - 外键约束和级联删除

### 🔧 Security Modules
- **新增安全模块**
  - `backend/pkg/crypto/security.go` - 安全管理器
    - `SecureWriteFile()` - 加密写入（加密 + HMAC + 0600）
    - `SecureReadFile()` - 安全读取（验证 HMAC + 解密）
    - `ComputeHMAC()` - 计算 HMAC-SHA256
    - `VerifyHMAC()` - 验证 HMAC
    - `SecureAllFiles()` - 批量设置文件权限

- **登录管理器增强**
  - 使用 SecurityManager 替代 KeyManager
  - 登录缓存加密存储（AES-256-GCM + HMAC）
  - 凭证导出/导入支持 HMAC 验证
  - 自动转换旧格式文件

### 🛠️ Migration Tool
- **数据迁移工具** (`backend/cmd/migrate/main.go`)
  - 自动备份 JSON 文件到 `~/.wemediaspider/backup/`
  - 解析所有 JSON 文件并去重
  - 批量插入数据库（事务保护）
  - 更新统计信息
  - 验证数据完整性
  - 生成迁移报告

### 📝 Documentation
- **新增文档**
  - `SECURITY.md` - 完整的安全架构文档
    - 加密存储说明
    - 完整性校验机制
    - 密钥管理方案
    - 文件权限管理
    - 安全威胁模型
    - 审计建议

### 🐛 Bug Fixes
- 修复导入历史数据界面滚动穿透问题
- 修复卡片标题垂直对齐问题
- 修复按钮右对齐布局问题
- 修复展开时标题不固定的问题

### 🔄 Breaking Changes
- **数据存储格式变更**: 从 JSON 文件迁移到 SQLite 数据库
  - 旧版本数据需要使用迁移工具转换
  - 运行 `go run backend/cmd/migrate/main.go` 进行迁移
  - 自动备份原始 JSON 文件

- **登录缓存格式变更**: 从明文/加密迁移到加密+HMAC
  - 自动检测并升级旧格式
  - 无需手动操作

### 📦 Technical Details
- **数据库**: SQLite 3 with GORM v1.25.12
- **加密**: AES-256-GCM + HMAC-SHA256
- **密钥派生**: PBKDF2 (100,000 iterations)
- **文件权限**: 0600 (owner read/write only)
- **批量操作**: 500 records per batch
- **连接池**: Max 25 connections, 5 idle

### 📋 Migration Guide
1. **备份数据**: 确保 `~/.wemediaspider/data/` 目录下的 JSON 文件已备份
2. **运行迁移**: `go run backend/cmd/migrate/main.go`
3. **验证数据**: 检查 `~/.wemediaspider/migration_report.txt`
4. **启动应用**: 正常启动应用，数据已迁移到数据库

### 🔒 Security Notes
- 所有敏感文件现在使用 0600 权限
- 登录凭证使用 AES-256-GCM 加密 + HMAC 完整性校验
- 主密钥使用 PBKDF2 派生，不直接存储
- 支持凭证导出/导入（.zgswx 格式，含 HMAC）

## [1.0.3] - 2026-03-09

### Changed
- 🎨 **界面优化**
  - 将检查更新按钮移至左侧边栏底部（收起按钮上方）
  - 边栏收起时仅显示图标，展开时显示"检查更新"文字
  - 从首页移除浮动的检查更新按钮
  - 优化按钮样式和交互体验

## [1.0.2] - 2026-03-09

### Added
- ✨ **更新检查增强**
  - 更新内容支持 Markdown 格式渲染（标题、列表、代码块、链接等）
  - 添加手动检查更新按钮（首页右下角）
  - 点击"稍后更新"后当天不再提醒
  - 手动检查更新时显示加载状态
  - 优化更新对话框样式和间距

### Fixed
- 🐛 **样式优化**
  - 完善 Markdown 内容样式（标题、列表、代码、表格等）
  - 优化更新对话框的可读性

## [1.0.1] - 2026-03-09

### Fixed
- 🐛 **版本更新检查修复**
  - 修复 GitHub API 返回 404 错误的问题
  - 添加必需的 User-Agent 和 Accept 请求头
  - 使用 http.NewRequest 替代 client.Get 以更好地控制请求
  - 现在可以正常检查和提示新版本更新

## [1.0.0] - 2026-03-09

### Added
- 🚀 **批量爬取功能**
  - 支持多个公众号并发爬取
  - 可配置并发工作线程数（1-10）
  - 实时进度显示和状态更新
  - 支持取消正在进行的爬取任务

- 📊 **多格式导出**
  - Excel (.xlsx) - 完整的文章数据表格
  - CSV (.csv) - 兼容性最好的表格格式
  - JSON (.json) - 结构化数据格式
  - Markdown (.md) - 适合阅读和编辑的文本格式

- 🖼️ **图片下载功能**
  - 从文章内容中自动提取图片
  - 批量下载图片到本地
  - 支持自定义保存目录
  - 实时下载进度显示
  - 可配置并发下载数

- 🔐 **安全特性**
  - 登录凭证 AES-256-GCM 加密存储
  - 支持凭证导出和导入（.zgswx 格式）
  - 自动清理过期缓存
  - 安全的本地数据存储

- 💾 **数据管理**
  - 自动保存爬取数据到本地
  - 支持数据导入（覆盖/追加模式）
  - 智能去重，避免重复数据
  - 历史数据文件列表查看
  - 数据文件删除功能

- 🔍 **搜索和筛选**
  - 公众号搜索功能
  - 按日期范围筛选文章
  - 按关键词筛选文章标题
  - 按网盘类型筛选文章

- 📈 **统计功能**
  - 总文章数统计
  - 今日文章数统计（每日自动重置）
  - 已爬取公众号数量
  - 图片下载数量统计
  - 最后爬取时间记录

- 🔄 **自动更新**
  - 自动检查 GitHub 最新版本
  - 显示版本更新日志
  - 一键跳转下载页面

- 🎨 **用户界面**
  - 现代化暗色主题
  - 响应式布局设计
  - 流畅的页面切换动画
  - 友好的操作提示
  - 优化的输入框渲染

- ⚙️ **配置管理**
  - 持久化配置存储
  - 默认配置恢复
  - 缓存管理功能
  - 缓存统计查看

### Technical Details
- **后端**: Go 1.18+ with Wails v2.11.0
- **前端**: React 18 + TypeScript 5 + Ant Design 5
- **状态管理**: Zustand
- **构建工具**: Vite 5
- **加密算法**: AES-256-GCM
- **数据格式**: JSON

### System Requirements
- **Windows**: Windows 10/11 (64-bit)
- **Memory**: 最低 4GB RAM
- **Disk Space**: 最低 100MB 可用空间
- **Network**: 需要网络连接以访问微信公众号平台

### Known Issues
- 首次启动可能需要较长时间加载
- 大量文章爬取时建议适当降低并发数
- 图片下载速度受网络环境影响

### Security
- 所有登录凭证均加密存储在本地
- 不会上传任何用户数据到第三方服务器
- 建议定期备份导出的凭证文件

[1.2.0]: https://github.com/vag-Zhao/WeMediaSpider-Go/compare/v1.0.3...v1.2.0
[1.0.3]: https://github.com/vag-Zhao/WeMediaSpider-Go/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/vag-Zhao/WeMediaSpider-Go/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/vag-Zhao/WeMediaSpider-Go/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/vag-Zhao/WeMediaSpider-Go/releases/tag/v1.0.0
