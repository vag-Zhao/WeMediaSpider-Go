import React, { useState, useEffect, useMemo } from 'react'
import {
  Card,
  Table,
  Button,
  Space,
  Input,
  Select,
  Modal,
  App,
  Tag,
  DatePicker,
  Dropdown,
  MenuProps,
  Pagination,
  Tabs,
  message as antdMessage,
  Checkbox,
  Progress,
  Spin,
  List,
  Empty,
  Tooltip,
  Collapse,
} from 'antd'
import {
  ExportOutlined,
  EyeOutlined,
  SearchOutlined,
  FilterOutlined,
  DownOutlined,
  CopyOutlined,
  LinkOutlined,
  ReadOutlined,
  PictureOutlined,
  DownloadOutlined,
  StopOutlined,
  ImportOutlined,
  FolderOpenOutlined,
  DeleteOutlined,
  FileTextOutlined,
  ClockCircleOutlined,
  TeamOutlined,
  CaretRightOutlined,
} from '@ant-design/icons'
import dayjs from 'dayjs'
import { useScrapeStore } from '../stores/scrapeStore'
import { api, events } from '../services/api'
import type { Article } from '../types'

const { Search } = Input
const { RangePicker } = DatePicker
const { Panel } = Collapse

// 网盘链接类型
interface CloudDriveLink {
  accountName: string
  title: string
  link: string
  articleLink: string
}

// 提取网盘链接的正则表达式
const cloudDrivePatterns = [
  /https?:\/\/pan\.baidu\.com\/s\/[a-zA-Z0-9_-]+/g,
  /https?:\/\/www\.aliyundrive\.com\/s\/[a-zA-Z0-9_-]+/g,
  /https?:\/\/pan\.quark\.cn\/s\/[a-zA-Z0-9_-]+/g,
  /https?:\/\/cloud\.189\.cn\/[a-zA-Z0-9_\-\/]+/g,
  /https?:\/\/www\.123pan\.com\/s\/[a-zA-Z0-9_-]+/g,
  /https?:\/\/[a-zA-Z0-9]+\.lanzou[a-z]\.com\/[a-zA-Z0-9_-]+/g,
  /https?:\/\/pan\.xunlei\.com\/s\/[a-zA-Z0-9_-]+/g,
  /https?:\/\/share\.weiyun\.com\/[a-zA-Z0-9_-]+/g,
  /https?:\/\/www\.ctfile\.com\/[a-zA-Z0-9_\-\/]+/g,
  /https?:\/\/onedrive\.live\.com\/[a-zA-Z0-9_\-\/?=&]+/g,
]

// 从文章内容中提取网盘链接
const extractCloudDriveLinks = (articles: Article[]): CloudDriveLink[] => {
  const links: CloudDriveLink[] = []

  articles.forEach(article => {
    const content = `${article.title} ${article.digest} ${article.content}`
    const foundLinks = new Set<string>()

    cloudDrivePatterns.forEach(pattern => {
      const matches = content.match(pattern)
      if (matches) {
        matches.forEach(link => foundLinks.add(link))
      }
    })

    foundLinks.forEach(link => {
      links.push({
        accountName: article.accountName,
        title: article.title,
        link: link,
        articleLink: article.link,
      })
    })
  })

  return links
}

const ResultsPage: React.FC = () => {
  const { message } = App.useApp()
  const { articles, setArticles } = useScrapeStore()
  const [filteredArticles, setFilteredArticles] = useState<Article[]>(articles || [])
  const [selectedArticle, setSelectedArticle] = useState<Article | null>(null)
  const [previewVisible, setPreviewVisible] = useState(false)
  const [exportLoading, setExportLoading] = useState(false)
  const [searchKeyword, setSearchKeyword] = useState('')
  const [selectedAccount, setSelectedAccount] = useState<string>()
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(
    null
  )
  const [pageSize, setPageSize] = useState(15)
  const [currentPage, setCurrentPage] = useState(1)
  const [activeTab, setActiveTab] = useState('articles')
  const [linkCurrentPage, setLinkCurrentPage] = useState(1)
  const [selectedArticles, setSelectedArticles] = useState<string[]>([])
  const [downloading, setDownloading] = useState(false)
  const [downloadProgress, setDownloadProgress] = useState({ current: 0, total: 0, message: '' })
  const [totalImages, setTotalImages] = useState(0)
  const [importModalVisible, setImportModalVisible] = useState(false)
  const [dataFiles, setDataFiles] = useState<any[]>([])
  const [loadingFiles, setLoadingFiles] = useState(false)
  const [loadingData, setLoadingData] = useState(false)

  // 提取网盘链接
  const cloudDriveLinks = useMemo(() => {
    return extractCloudDriveLinks(filteredArticles)
  }, [filteredArticles])

  // 获取当前页的网盘链接数据
  const currentLinkPageData = cloudDriveLinks.slice(
    (linkCurrentPage - 1) * pageSize,
    linkCurrentPage * pageSize
  )

  // 复制所有网盘链接
  const handleCopyAllLinks = () => {
    const text = cloudDriveLinks
      .map((item, index) => `${index + 1}.${item.title}\n链接：${item.link}`)
      .join('\n')

    navigator.clipboard.writeText(text).then(() => {
      message.success(`已复制 ${cloudDriveLinks.length} 个链接`)
    }).catch(() => {
      message.error('复制失败')
    })
  }

  // 监听图片下载事件
  useEffect(() => {
    const handleProgress = (progress: any) => {
      setDownloadProgress({
        current: progress.current,
        total: progress.total,
        message: progress.message,
      })
    }

    const handleCompleted = (data: any) => {
      message.success(`图片下载完成！共下载 ${data.total} 张图片`)
      setDownloading(false)
      setDownloadProgress({ current: 0, total: 0, message: '' })
      setSelectedArticles([])
    }

    const handleError = (error: any) => {
      const errorMsg = error?.error || error?.message || String(error)
      if (!errorMsg.includes('context canceled') && !errorMsg.includes('canceled')) {
        message.error('下载失败: ' + errorMsg)
      }
      setDownloading(false)
      setDownloadProgress({ current: 0, total: 0, message: '' })
    }

    const unsubProgress = events.onImageProgress(handleProgress)
    const unsubCompleted = events.onImageCompleted(handleCompleted)
    const unsubError = events.onImageError(handleError)

    return () => {
      events.offImageProgress(unsubProgress)
      events.offImageCompleted(unsubCompleted)
      events.offImageError(unsubError)
    }
  }, [])

  // 批量下载选中文章的图片
  const handleBatchDownloadImages = async () => {
    if (selectedArticles.length === 0) {
      message.warning('请先选择文章')
      return
    }

    try {
      // 提取所有选中文章的图片
      let allImages: any[] = []
      let imageIndex = 1

      for (const articleId of selectedArticles) {
        const article = filteredArticles.find(a => a.id === articleId)
        if (article && article.content) {
          const images = await api.extractArticleImages(article.content)
          images.forEach((img: any) => {
            allImages.push({
              url: img.url,
              index: imageIndex++,
              filename: `${imageIndex - 1}${img.filename.substring(img.filename.lastIndexOf('.'))}`,
              articleTitle: article.title,
              accountName: article.accountName,
            })
          })
        }
      }

      if (allImages.length === 0) {
        message.warning('选中的文章没有图片')
        return
      }

      setTotalImages(allImages.length)

      // 选择保存目录
      const dir = await api.selectDirectory()
      if (!dir) {
        message.info('已取消下载')
        return
      }

      // 开始下载
      setDownloading(true)
      setDownloadProgress({ current: 0, total: allImages.length, message: '准备下载...' })

      // 使用 5 个并发下载
      await api.batchDownloadImages(allImages, dir, 5)
    } catch (error: any) {
      const errorMsg = error?.message || '未知错误'
      if (errorMsg.includes('用户取消')) {
        message.info('已取消下载')
      } else {
        message.error('下载失败: ' + errorMsg)
      }
      setDownloading(false)
      setDownloadProgress({ current: 0, total: 0, message: '' })
    }
  }

  // 取消下载
  const handleCancelDownload = async () => {
    try {
      await api.cancelImageDownload()
      message.info('已取消下载')
      setDownloading(false)
      setDownloadProgress({ current: 0, total: 0, message: '' })
    } catch (error: any) {
      message.error('取消失败: ' + (error.message || '未知错误'))
    }
  }

  // 打开导入数据对话框
  const handleOpenImportModal = async () => {
    setImportModalVisible(true)
    await loadDataFileList()
  }

  // 加载数据文件列表
  const loadDataFileList = async () => {
    try {
      setLoadingFiles(true)
      const files = await (window as any).go.app.App.ListDataFiles()
      // 为每个文件加载文章列表
      const filesWithArticles = await Promise.all(
        (files || []).map(async (file: any) => {
          try {
            const articles = await (window as any).go.app.App.LoadDataFile(file.filepath)
            return { ...file, articles: articles || [] }
          } catch (error) {
            console.error('Failed to load articles for file:', file.filepath, error)
            return { ...file, articles: [] }
          }
        })
      )
      setDataFiles(filesWithArticles)
    } catch (error: any) {
      message.error('加载文件列表失败: ' + (error.message || '未知错误'))
    } finally {
      setLoadingFiles(false)
    }
  }

  // 导入数据文件
  const handleImportData = async (filepath: string, mode: 'replace' | 'append') => {
    try {
      setLoadingData(true)
      const loadedArticles = await (window as any).go.app.App.LoadDataFile(filepath)

      if (mode === 'replace') {
        // 覆盖导入
        setArticles(loadedArticles)
        setFilteredArticles(loadedArticles)
        message.success(`成功导入 ${loadedArticles.length} 篇文章（覆盖模式）`)
      } else {
        // 追加导入
        const existingIds = new Set(articles.map(a => a.id))
        const newArticles = loadedArticles.filter((a: Article) => !existingIds.has(a.id))
        const mergedArticles = [...articles, ...newArticles]
        setArticles(mergedArticles)
        setFilteredArticles(mergedArticles)
        message.success(`成功追加 ${newArticles.length} 篇新文章，跳过 ${loadedArticles.length - newArticles.length} 篇重复文章`)
      }

      setImportModalVisible(false)
    } catch (error: any) {
      const errorMsg = error?.message || '未知错误'
      if (errorMsg.includes('用户取消')) {
        message.info('已取消导入')
      } else {
        message.error('导入失败: ' + errorMsg)
      }
    } finally {
      setLoadingData(false)
    }
  }

  // 删除数据文件
  const handleDeleteDataFile = async (filepath: string) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除这个数据文件吗？此操作不可恢复。',
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await (window as any).go.app.App.DeleteDataFile(filepath)
          message.success('删除成功')
          await loadDataFileList()
        } catch (error: any) {
          message.error('删除失败: ' + (error.message || '未知错误'))
        }
      },
    })
  }

  // 计算自适应的每页条数
  useEffect(() => {
    const calculatePageSize = () => {
      // 表格行高约 39px (small size)
      const rowHeight = 39
      // Card标题高度约55px, 过滤器高度约40px, Card padding 24px, 表头39px, 翻页组件40px, 顶部间距4px
      const fixedHeight = 55 + 40 + 24 + 39 + 40 + 4
      // 可用高度 = 视口高度 - 固定高度
      const availableHeight = window.innerHeight - fixedHeight
      // 计算可显示的行数，并额外加1行
      const rows = Math.floor(availableHeight / rowHeight) + 1
      // 至少显示 6 行，最多 51 行
      setPageSize(Math.max(6, Math.min(rows, 51)))
    }

    calculatePageSize()
    window.addEventListener('resize', calculatePageSize)
    return () => window.removeEventListener('resize', calculatePageSize)
  }, [])

  // 获取当前页数据
  const currentPageData = filteredArticles.slice(
    (currentPage - 1) * pageSize,
    currentPage * pageSize
  )

  // 获取所有公众号列表
  const accountNames = Array.from(
    new Set((articles || []).map((a) => a.accountName))
  ).sort()

  // 应用过滤
  const applyFilters = () => {
    let filtered = [...(articles || [])]

    // 关键词过滤
    if (searchKeyword) {
      filtered = filtered.filter(
        (a) =>
          a.title.includes(searchKeyword) ||
          a.digest.includes(searchKeyword)
      )
    }

    // 公众号过滤
    if (selectedAccount) {
      filtered = filtered.filter((a) => a.accountName === selectedAccount)
    }

    // 日期过滤
    if (dateRange) {
      const [start, end] = dateRange
      filtered = filtered.filter((a) => {
        const publishDate = dayjs(a.publishTime)
        return (
          publishDate.isAfter(start.startOf('day')) &&
          publishDate.isBefore(end.endOf('day'))
        )
      })
    }

    setFilteredArticles(filtered)
    setCurrentPage(1) // 重置到第一页
  }

  // 重置过滤
  const resetFilters = () => {
    setSearchKeyword('')
    setSelectedAccount(undefined)
    setDateRange(null)
    setFilteredArticles(articles || [])
    setCurrentPage(1) // 重置到第一页
  }

  // 预览文章
  const handlePreview = (article: Article) => {
    setSelectedArticle(article)
    setPreviewVisible(true)
  }

  // 导出文章
  const handleExport = async (format: string) => {
    try {
      setExportLoading(true)

      // 文件扩展名映射
      const extensionMap: { [key: string]: string } = {
        'excel': 'xlsx',
        'csv': 'csv',
        'json': 'json',
        'markdown': 'md'
      }

      const extension = extensionMap[format] || format
      const defaultFilename = `articles_${dayjs().format('YYYYMMDD_HHmmss')}.${extension}`

      // 使用正确的 FileFilter 类型
      const filterNameMap: { [key: string]: string } = {
        'excel': 'Excel 文件 (*.xlsx)',
        'csv': 'CSV 文件 (*.csv)',
        'json': 'JSON 文件 (*.json)',
        'markdown': 'Markdown 文件 (*.md)'
      }

      const filters = [
        {
          DisplayName: filterNameMap[format] || format.toUpperCase(),
          Pattern: `*.${extension}`,
        },
      ]

      const filename = await api.selectSaveFile(defaultFilename, filters)
      if (!filename) {
        setExportLoading(false)
        message.info('已取消导出')
        return
      }

      // 转换文章数组为 models.Article 类型
      await api.exportArticles(filteredArticles as any, format, filename)
      message.success(`成功导出 ${filteredArticles.length} 篇文章`)
    } catch (error: any) {
      const errorMsg = error?.message || '未知错误'
      if (errorMsg.includes('用户取消')) {
        message.info('已取消导出')
      } else {
        message.error('导出失败: ' + errorMsg)
      }
    } finally {
      setExportLoading(false)
    }
  }

  // 导出菜单项
  const exportMenuItems: MenuProps['items'] = [
    {
      key: 'csv',
      label: 'CSV 格式',
      onClick: () => handleExport('csv'),
    },
    {
      key: 'json',
      label: 'JSON 格式',
      onClick: () => handleExport('json'),
    },
    {
      key: 'excel',
      label: 'Excel 格式',
      onClick: () => handleExport('excel'),
    },
    {
      key: 'markdown',
      label: 'Markdown 格式',
      onClick: () => handleExport('markdown'),
    },
  ]

  // 表格列定义
  const columns = [
    {
      title: (
        <Checkbox
          checked={selectedArticles.length === filteredArticles.length && filteredArticles.length > 0}
          indeterminate={selectedArticles.length > 0 && selectedArticles.length < filteredArticles.length}
          onChange={(e) => {
            if (e.target.checked) {
              setSelectedArticles(filteredArticles.map(a => a.id))
            } else {
              setSelectedArticles([])
            }
          }}
        />
      ),
      key: 'selection',
      width: 50,
      render: (_: any, record: Article) => (
        <Checkbox
          checked={selectedArticles.includes(record.id)}
          onChange={(e) => {
            if (e.target.checked) {
              setSelectedArticles([...selectedArticles, record.id])
            } else {
              setSelectedArticles(selectedArticles.filter(id => id !== record.id))
            }
          }}
        />
      ),
    },
    {
      title: '公众号',
      dataIndex: 'accountName',
      key: 'accountName',
      width: 70,
      render: (name: string) => <Tag color="blue">{name}</Tag>,
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      width: 200,
      ellipsis: true,
      render: (title: string, record: Article) => (
        <a onClick={() => handlePreview(record)}>{title}</a>
      ),
    },
    {
      title: '发布时间',
      dataIndex: 'publishTime',
      key: 'publishTime',
      width: 140,
    },
  ]

  // 网盘链接表格列定义
  const linkColumns = [
    {
      title: '序号',
      key: 'index',
      width: 60,
      render: (_: any, __: any, index: number) => (linkCurrentPage - 1) * pageSize + index + 1,
    },
    {
      title: '公众号',
      dataIndex: 'accountName',
      key: 'accountName',
      width: 100,
      render: (name: string) => <Tag color="blue">{name}</Tag>,
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      width: 250,
      ellipsis: true,
      render: (title: string, record: CloudDriveLink) => (
        <a href={record.articleLink} target="_blank" rel="noopener noreferrer">
          {title}
        </a>
      ),
    },
    {
      title: '网盘链接',
      dataIndex: 'link',
      key: 'link',
      ellipsis: true,
      render: (link: string) => (
        <a href={link} target="_blank" rel="noopener noreferrer">
          {link}
        </a>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: any, record: CloudDriveLink, index: number) => (
        <Button
          size="small"
          icon={<CopyOutlined />}
          onClick={() => {
            const globalIndex = (linkCurrentPage - 1) * pageSize + index + 1
            const text = `${globalIndex}.${record.title}\n链接：${record.link}`
            navigator.clipboard.writeText(text)
            message.success('已复制')
          }}
        >
          复制
        </Button>
      ),
    },
  ]

  return (
    <div style={{ height: '100%', overflow: 'hidden' }}>
      <Card
        title={
          <Tabs
            activeKey={activeTab}
            onChange={setActiveTab}
            items={[
              {
                key: 'articles',
                label: (
                  <span>
                    <ReadOutlined /> 文章列表 ({filteredArticles.length}/{(articles || []).length})
                  </span>
                ),
              },
              {
                key: 'links',
                label: (
                  <span>
                    <LinkOutlined /> 网盘链接 ({cloudDriveLinks.length})
                  </span>
                ),
              },
            ]}
            size="small"
          />
        }
        style={{ boxShadow: '0 2px 8px rgba(0,0,0,0.3)', height: '100%', display: 'flex', flexDirection: 'column' }}
        styles={{ body: { flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column', padding: 12 } }}
        extra={
          activeTab === 'articles' ? (
            <Space>
              <Button
                icon={<ImportOutlined />}
                onClick={handleOpenImportModal}
                size="small"
              >
                导入数据
              </Button>
              <Dropdown menu={{ items: exportMenuItems }} placement="bottomRight">
                <Button
                  icon={<ExportOutlined />}
                  loading={exportLoading}
                  size="small"
                >
                  导出 <DownOutlined />
                </Button>
              </Dropdown>
            </Space>
          ) : (
            <Button
              type="primary"
              icon={<CopyOutlined />}
              onClick={handleCopyAllLinks}
              size="small"
              disabled={cloudDriveLinks.length === 0}
            >
              复制全部链接
            </Button>
          )
        }
      >
        {activeTab === 'articles' ? (
          <>
            {/* 过滤器 */}
            <div style={{ marginBottom: 8, display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 8 }}>
              <Space size="small" wrap>
                <Search
                  placeholder="搜索"
                  value={searchKeyword}
                  onChange={(e) => setSearchKeyword(e.target.value)}
                  onSearch={applyFilters}
                  style={{ width: 200 }}
                  size="small"
                />

                <Select
                  placeholder="公众号"
                  value={selectedAccount}
                  onChange={setSelectedAccount}
                  style={{ width: 140 }}
                  allowClear
                  size="small"
                >
                  {accountNames.map((name) => (
                    <Select.Option key={name} value={name}>
                      {name}
                    </Select.Option>
                  ))}
                </Select>

                <RangePicker
                  value={dateRange}
                  onChange={(dates) => setDateRange(dates as any)}
                  style={{ width: 220 }}
                  size="small"
                />

                <Button icon={<FilterOutlined />} onClick={applyFilters} type="primary" size="small">
                  过滤
                </Button>

                <Button onClick={resetFilters} size="small">重置</Button>
              </Space>

              {selectedArticles.length > 0 && (
                <Button
                  type="primary"
                  icon={<DownloadOutlined />}
                  onClick={handleBatchDownloadImages}
                  size="small"
                  disabled={downloading}
                >
                  下载图片 ({selectedArticles.length})
                </Button>
              )}
            </div>

            {/* 文章表格 */}
            <div style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
              <Table
                dataSource={currentPageData}
                columns={columns}
                rowKey="id"
                size="small"
                pagination={false}
                scroll={{ y: `calc(100vh - 167px)` }}
              />
            </div>

            {/* 固定在底部的翻页组件 */}
            <div style={{
              paddingTop: 4,
              display: 'flex',
              justifyContent: 'center',
              alignItems: 'center',
              flexShrink: 0
            }}>
              <Pagination
                current={currentPage}
                pageSize={pageSize}
                total={filteredArticles.length}
                onChange={(page) => setCurrentPage(page)}
                size="small"
                showSizeChanger={false}
                showTotal={(total) => `共 ${total} 篇`}
              />
            </div>
          </>
        ) : (
          <>
            {/* 网盘链接表格 */}
            <div style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
              {cloudDriveLinks.length > 0 ? (
                <>
                  <Table
                    dataSource={currentLinkPageData}
                    columns={linkColumns}
                    rowKey={(record, index) => `${record.link}-${index}`}
                    size="small"
                    pagination={false}
                    scroll={{ y: `calc(100vh - 167px)` }}
                  />
                </>
              ) : (
                <div style={{
                  flex: 1,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: '#999',
                  fontSize: 14,
                }}>
                  未检测到网盘链接
                </div>
              )}
            </div>

            {/* 固定在底部的翻页组件 */}
            {cloudDriveLinks.length > 0 && (
              <div style={{
                paddingTop: 4,
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
                flexShrink: 0
              }}>
                <Pagination
                  current={linkCurrentPage}
                  pageSize={pageSize}
                  total={cloudDriveLinks.length}
                  onChange={(page) => setLinkCurrentPage(page)}
                  size="small"
                  showSizeChanger={false}
                  showTotal={(total) => `共 ${total} 个链接`}
                />
              </div>
            )}
          </>
        )}
      </Card>

      {/* 文章预览对话框 */}
      <Modal
        title={selectedArticle?.title}
        open={previewVisible}
        onCancel={() => setPreviewVisible(false)}
        width={900}
        centered
        closable={false}
        styles={{ body: { padding: 0, maxHeight: 'calc(100vh - 200px)', display: 'flex', flexDirection: 'column' } }}
        footer={[
          <Button key="close" onClick={() => setPreviewVisible(false)}>
            关闭
          </Button>,
          <Button
            key="open"
            type="primary"
            href={selectedArticle?.link}
            target="_blank"
          >
            打开原文
          </Button>,
        ]}
      >
        {selectedArticle && (
          <>
            <div style={{
              padding: '12px 24px',
              borderBottom: '1px solid #303030',
              display: 'flex',
              gap: 24,
              flexWrap: 'wrap',
              fontSize: 14,
              color: '#999',
              position: 'sticky',
              top: 0,
              zIndex: 1
            }}>
              <span>公众号: {selectedArticle.accountName}</span>
              <span>发布时间: {selectedArticle.publishTime}</span>
            </div>

            <div style={{ padding: '24px 48px', overflow: 'auto', flex: 1 }}>
              {selectedArticle.digest && (
                <div style={{
                  padding: 12,
                  background: '#1a1a1a',
                  borderRadius: 4,
                  marginBottom: 16,
                  color: '#999',
                  fontSize: 14,
                  lineHeight: 1.6
                }}>
                  {selectedArticle.digest}
                </div>
              )}

              {selectedArticle.content && (
                <div style={{
                  padding: 16,
                  background: '#0d0d0d',
                  borderRadius: 4,
                  whiteSpace: 'pre-wrap',
                  lineHeight: 1.8,
                  fontSize: 15
                }}>
                  {selectedArticle.content}
                </div>
              )}
            </div>
          </>
        )}
      </Modal>

      {/* 图片下载遮罩层 */}
      {downloading && (
        <div style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          background: 'rgba(0, 0, 0, 0.85)',
          backdropFilter: 'blur(8px)',
          zIndex: 9999,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          animation: 'fadeIn 0.3s ease-in-out',
        }}>
          <div style={{
            background: 'linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%)',
            borderRadius: 16,
            padding: 48,
            minWidth: 500,
            boxShadow: '0 20px 60px rgba(0, 0, 0, 0.5)',
            border: '1px solid rgba(255, 255, 255, 0.1)',
            animation: 'slideUp 0.4s ease-out',
          }}>
            <div style={{
              textAlign: 'center',
              marginBottom: 32,
            }}>
              <DownloadOutlined style={{
                fontSize: 48,
                color: '#07C160',
                marginBottom: 16,
                animation: 'pulse 2s ease-in-out infinite',
              }} />
              <h2 style={{
                fontSize: 24,
                fontWeight: 600,
                margin: 0,
                marginBottom: 8,
                color: '#fff',
              }}>
                正在下载图片
              </h2>
              <p style={{
                fontSize: 14,
                color: '#999',
                margin: 0,
              }}>
                {downloadProgress.message}
              </p>
            </div>

            <div style={{ marginBottom: 24 }}>
              <Progress
                percent={downloadProgress.total > 0 ? Math.round((downloadProgress.current / downloadProgress.total) * 100) : 0}
                status="active"
                strokeColor={{
                  '0%': '#07C160',
                  '100%': '#06AE56',
                }}
                trailColor="rgba(255, 255, 255, 0.1)"
                strokeWidth={12}
                format={(percent) => (
                  <span style={{ color: '#fff', fontSize: 16, fontWeight: 600 }}>
                    {percent}%
                  </span>
                )}
              />
              <div style={{
                display: 'flex',
                justifyContent: 'space-between',
                marginTop: 12,
                fontSize: 13,
                color: '#999',
              }}>
                <span>{downloadProgress.current} / {downloadProgress.total} 张</span>
                <span>{downloadProgress.total - downloadProgress.current} 张待下载</span>
              </div>
            </div>

            <Button
              danger
              block
              size="large"
              icon={<StopOutlined />}
              onClick={handleCancelDownload}
              style={{
                height: 48,
                fontSize: 16,
                fontWeight: 500,
                borderRadius: 8,
              }}
            >
              取消下载
            </Button>
          </div>

          <style>{`
            @keyframes fadeIn {
              from {
                opacity: 0;
              }
              to {
                opacity: 1;
              }
            }

            @keyframes slideUp {
              from {
                transform: translateY(30px);
                opacity: 0;
              }
              to {
                transform: translateY(0);
                opacity: 1;
              }
            }

            @keyframes pulse {
              0%, 100% {
                transform: scale(1);
                opacity: 1;
              }
              50% {
                transform: scale(1.1);
                opacity: 0.8;
              }
            }
          `}</style>
        </div>
      )}

      {/* 导入数据对话框 */}
      <Modal
        title={
          <Space>
            <ImportOutlined style={{ color: '#07C160' }} />
            <span>导入历史数据</span>
          </Space>
        }
        open={importModalVisible}
        onCancel={() => setImportModalVisible(false)}
        width={900}
        centered
        footer={null}
        styles={{ body: { maxHeight: 'calc(100vh - 300px)', overflow: 'auto' } }}
      >
        <style>{`
          /* 固定 Collapse 标题位置 */
          .ant-collapse-item > .ant-collapse-header {
            position: sticky !important;
            top: 0 !important;
            z-index: 10 !important;
            background: #1a1a1a !important;
            display: flex !important;
            align-items: center !important;
            width: 100% !important;
          }

          /* 展开时保持标题固定 */
          .ant-collapse-item-active > .ant-collapse-header {
            position: sticky !important;
            top: 0 !important;
            z-index: 10 !important;
          }

          /* 确保标题内容区域占满剩余空间 */
          .ant-collapse-header-text {
            flex: 1 !important;
            display: flex !important;
            align-items: center !important;
            width: 100% !important;
          }
        `}</style>
        {loadingFiles ? (
          <div style={{ textAlign: 'center', padding: 40 }}>
            <Spin size="large" />
          </div>
        ) : dataFiles.length === 0 ? (
          <Empty
            description="暂无历史数据"
            style={{ padding: 40 }}
          />
        ) : (
          <Collapse
            bordered={false}
            expandIcon={({ isActive }) => <CaretRightOutlined rotate={isActive ? 90 : 0} />}
            style={{ background: 'transparent' }}
            collapsible="header"
          >
            {dataFiles.map((file: any) => (
              <Panel
                key={file.filepath}
                header={
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10, width: '100%' }}>
                    {/* 图标 */}
                    <div style={{
                      width: 36,
                      height: 36,
                      borderRadius: 6,
                      background: 'linear-gradient(135deg, rgba(7, 193, 96, 0.2) 0%, rgba(7, 193, 96, 0.1) 100%)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      border: '1px solid rgba(7, 193, 96, 0.3)',
                      flexShrink: 0,
                    }}>
                      <FileTextOutlined style={{ fontSize: 16, color: '#07C160' }} />
                    </div>

                    {/* 内容区 */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <span style={{ fontSize: 14, fontWeight: 500, lineHeight: '20px' }}>{file.filename}</span>
                      <span style={{ fontSize: 12, color: '#999', lineHeight: '20px', display: 'flex', alignItems: 'center', gap: 4 }}>
                        <ClockCircleOutlined style={{ fontSize: 12 }} /> <span>{file.saveTime}</span>
                      </span>
                      <Tag color="blue" style={{ fontSize: 11, padding: '0 6px', lineHeight: '18px', margin: 0 }}>
                        {file.totalCount} 篇
                      </Tag>
                    </div>

                    {/* 操作按钮 */}
                    <div style={{ display: 'flex', gap: 6, flexShrink: 0, marginLeft: 'auto' }} onClick={(e) => e.stopPropagation()}>
                      <Button
                        type="primary"
                        size="small"
                        icon={<ImportOutlined />}
                        onClick={(e) => {
                          e.stopPropagation()
                          handleImportData(file.filepath, 'replace')
                        }}
                        loading={loadingData}
                      >
                        覆盖导入
                      </Button>
                      <Button
                        size="small"
                        icon={<ImportOutlined />}
                        onClick={(e) => {
                          e.stopPropagation()
                          handleImportData(file.filepath, 'append')
                        }}
                        loading={loadingData}
                      >
                        追加导入
                      </Button>
                      <Button
                        danger
                        size="small"
                        icon={<DeleteOutlined />}
                        onClick={(e) => {
                          e.stopPropagation()
                          handleDeleteDataFile(file.filepath)
                        }}
                      >
                        删除
                      </Button>
                    </div>
                  </div>
                }
                style={{
                  marginBottom: 8,
                  background: '#1a1a1a',
                  borderRadius: 8,
                  border: '1px solid rgba(255, 255, 255, 0.1)',
                }}
              >
                {/* 展开后显示文章列表 */}
                <div style={{ paddingLeft: 60 }}>
                  {file.articles && file.articles.length > 0 && (
                    <div
                      style={{
                        maxHeight: '400px',
                        overflowY: 'auto',
                        paddingRight: 8,
                        // 自定义滚动条样式
                        scrollbarWidth: 'thin',
                        scrollbarColor: '#333 #1a1a1a',
                      }}
                      onWheel={(e) => {
                        const target = e.currentTarget
                        const atTop = target.scrollTop === 0
                        const atBottom = target.scrollTop + target.clientHeight >= target.scrollHeight - 1

                        // 如果在顶部向上滚动，或在底部向下滚动，阻止事件冒泡
                        if ((atTop && e.deltaY < 0) || (atBottom && e.deltaY > 0)) {
                          e.stopPropagation()
                        }
                      }}
                    >
                      <style>{`
                        div::-webkit-scrollbar {
                          width: 6px;
                        }
                        div::-webkit-scrollbar-track {
                          background: #1a1a1a;
                          border-radius: 3px;
                        }
                        div::-webkit-scrollbar-thumb {
                          background: #333;
                          border-radius: 3px;
                        }
                        div::-webkit-scrollbar-thumb:hover {
                          background: #444;
                        }
                      `}</style>
                      <List
                        size="small"
                        dataSource={file.articles}
                        renderItem={(article: Article) => (
                          <List.Item
                            style={{
                              padding: '8px 12px',
                              background: '#0d0d0d',
                              marginBottom: 4,
                              borderRadius: 4,
                              border: '1px solid rgba(255, 255, 255, 0.05)',
                            }}
                          >
                            <div style={{ flex: 1, minWidth: 0, display: 'flex', alignItems: 'center', gap: 8 }}>
                              <Tag color="blue" style={{ fontSize: 11, padding: '0 4px', lineHeight: '16px', margin: 0, flexShrink: 0 }}>
                                {article.accountName}
                              </Tag>
                              <span style={{ fontSize: 13, color: '#fff', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}>
                                {article.title}
                              </span>
                              <span style={{ fontSize: 11, color: '#666', flexShrink: 0, marginLeft: 'auto' }}>
                                {article.publishTime}
                              </span>
                            </div>
                          </List.Item>
                        )}
                      />
                    </div>
                  )}
                </div>
              </Panel>
            ))}
          </Collapse>
        )}
      </Modal>
    </div>
  )
}

export default ResultsPage
