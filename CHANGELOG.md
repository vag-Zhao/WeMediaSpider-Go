# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[1.0.3]: https://github.com/vag-Zhao/WeMediaSpider-Go/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/vag-Zhao/WeMediaSpider-Go/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/vag-Zhao/WeMediaSpider-Go/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/vag-Zhao/WeMediaSpider-Go/releases/tag/v1.0.0
