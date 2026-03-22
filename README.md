# WeMediaSpider-Go

微信公众号文章智能爬虫 - 支持批量爬取、多格式导出、数据库存储、专业级安全架构

[![Version](https://img.shields.io/badge/version-2.0.0-blue.svg)](https://github.com/vag-Zhao/WeMediaSpider-Go/releases)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.18+-00ADD8.svg)](https://golang.org/)
[![Wails](https://img.shields.io/badge/Wails-v2-red.svg)](https://wails.io/)

## 📸 应用截图

![应用界面](wx.png)
## 🚀 核心功能

- **批量爬取**: 多公众号并发爬取，实时进度显示
- **定时任务**: 可视化配置定时爬取，支持每天/每周/间隔执行
- **数据分析**: 文章时间分布图表、关键词词云
- **实时日志**: 浮动日志面板，实时查看运行状态
- **多格式导出**: Excel、CSV、JSON、Markdown
- **图片下载**: 批量下载文章图片
- **数据库存储**: SQLite + GORM，高效查询
- **安全加密**: AES-256-GCM + HMAC 完整性校验
- **智能缓存**: 避免重复请求

## 📦 快速开始

### 下载使用（推荐）

1. 访问 [Releases 页面](https://github.com/vag-Zhao/WeMediaSpider-Go/releases)
2. 下载 
3. 解压后运行 

### 从旧版本升级

**重要**: v2.0.0 包含架构重构，建议全新安装。

```bash
# 1. 备份数据（推荐）
cp -r ~/.wemediaspider ~/.wemediaspider.backup

# 2. 运行迁移工具
go run backend/cmd/migrate/main.go

# 3. 启动应用
```

## 🛠️ 开发

### 环境要求

- Go 1.18+
- Node.js 16+
- Wails CLI

### 开发模式

```bash
# 克隆项目
git clone https://github.com/vag-Zhao/WeMediaSpider-Go.git
cd WeMediaSpider-Go

# 运行开发模式
wails dev
```

### 构建应用

```bash
wails build
```

## 🗄️ 数据库架构

- **accounts** - 公众号表
- **articles** - 文章表（外键关联 accounts）
- **app_stats** - 应用统计表

数据库位置: `~/.wemediaspider/wemedia.db`

## 📝 技术栈

**后端**: Go + Wails v2 + SQLite + GORM
**前端**: React + TypeScript + Ant Design + Vite
**安全**: AES-256-GCM + HMAC-SHA256 + PBKDF2

## 📋 项目结构

```
WeMediaSpider/
├── backend/              # Go 后端
│   ├── app/             # 应用逻辑
│   ├── cmd/migrate/     # 数据迁移工具
│   ├── internal/        # 内部包
│   │   ├── database/    # 数据库模块
│   │   ├── repository/  # 数据访问层
│   │   └── spider/      # 爬虫核心
│   └── pkg/             # 公共包
├── frontend/            # React 前端
│   └── src/
│       ├── pages/       # 页面
│       └── components/  # 组件
└── wails.json          # Wails 配置
```

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE)

## 🙏 致谢

感谢所有用户的支持！

## 📞 反馈

- [Issues](https://github.com/vag-Zhao/WeMediaSpider-Go/issues)
- [Releases](https://github.com/vag-Zhao/WeMediaSpider-Go/releases)
- [Changelog](CHANGELOG.md)
