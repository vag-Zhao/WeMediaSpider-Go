import React, { useState, useEffect } from 'react'
import { Card, Row, Col, Button, Space, Statistic, Tag, App, Spin, Progress, Divider, Modal, Alert } from 'antd'
import {
  LoginOutlined,
  LogoutOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  WechatOutlined,
  FileTextOutlined,
  CloudDownloadOutlined,
  FireOutlined,
  ThunderboltOutlined,
  ExportOutlined,
  ImportOutlined,
  PictureOutlined,
  RocketOutlined,
  SafetyOutlined,
  ClockCircleOutlined,
  TrophyOutlined,
  TeamOutlined,
  GiftOutlined,
} from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { useScrapeStore } from '../stores/scrapeStore'
import { useLoginStore } from '../stores/loginStore'
import { api } from '../services/api'
import dayjs from 'dayjs'
import '../components/WaveAnimation.css'

const HomePage: React.FC = () => {
  const { message } = App.useApp()
  const navigate = useNavigate()
  const { articles } = useScrapeStore()
  const { loginStatus, setLoginStatus } = useLoginStore()
  const [loading, setLoading] = useState(false)
  const [checking, setChecking] = useState(true)
  const [appData, setAppData] = useState<any>({})
  const [updateInfo, setUpdateInfo] = useState<any>(null)
  const [showUpdateModal, setShowUpdateModal] = useState(false)

  // 检查更新
  const checkForUpdates = async () => {
    try {
      const versionInfo = await api.checkForUpdates()
      if (versionInfo.hasUpdate) {
        setUpdateInfo(versionInfo)
        setShowUpdateModal(true)
      }
    } catch (error) {
      console.error('检查更新失败:', error)
    }
  }

  // 检查登录状态和加载应用数据
  const checkLoginStatus = async () => {
    try {
      setChecking(true)
      if (typeof (window as any).go === 'undefined') {
        return
      }
      const status = await api.getLoginStatus()
      setLoginStatus(status)
      const data = await api.getAppData()
      setAppData(data)

      // 检查更新
      checkForUpdates()
    } catch (error) {
      console.error('Failed to check status:', error)
    } finally {
      setChecking(false)
    }
  }

  useEffect(() => {
    const timer = setTimeout(() => {
      checkLoginStatus()
    }, 100)
    return () => clearTimeout(timer)
  }, [])

  // 登录
  const handleLogin = async () => {
    try {
      setLoading(true)
      message.info('正在打开浏览器，请扫码登录...')
      await api.login()
      message.success('登录成功！')
      await checkLoginStatus()
    } catch (error: any) {
      if (error?.message?.includes('浏览器已关闭') || error?.message?.includes('登录已取消')) {
        message.info('登录已取消')
      } else {
        message.error('登录失败: ' + (error?.message || '未知错误'))
      }
    } finally {
      setLoading(false)
    }
  }

  // 退出登录
  const handleLogout = async () => {
    try {
      setLoading(true)
      await api.logout()
      message.success('已退出登录')
      await checkLoginStatus()
    } catch (error: any) {
      message.error('退出失败: ' + (error.message || '未知错误'))
    } finally {
      setLoading(false)
    }
  }

  // 导出凭证
  const handleExportCredentials = async () => {
    try {
      setLoading(true)
      const filepath = await api.exportCredentials()
      message.success(`凭证已导出到: ${filepath}`)
    } catch (error: any) {
      const errorMsg = error?.message || error?.toString() || '未知错误'
      if (errorMsg.includes('用户取消') || errorMsg.includes('取消操作')) {
        message.info('已取消导出')
      } else {
        message.error('导出失败: ' + errorMsg)
      }
    } finally {
      setLoading(false)
    }
  }

  // 导入凭证
  const handleImportCredentials = async () => {
    try {
      setLoading(true)
      await api.importCredentials()
      message.success('凭证导入成功！')
      await checkLoginStatus()
    } catch (error: any) {
      const errorMsg = error?.message || error?.toString() || '未知错误'
      if (errorMsg.includes('用户取消') || errorMsg.includes('取消操作')) {
        message.info('已取消导入')
      } else {
        message.error('导入失败: ' + errorMsg)
      }
    } finally {
      setLoading(false)
    }
  }

  // 计算统计数据 - 优先使用持久化数据
  const totalArticles = appData.totalArticles || articles?.length || 0
  const accountCount = appData.totalAccounts || new Set(articles?.map(a => a.accountName)).size || 0
  const todayArticles = appData.todayArticles || 0
  const totalImages = appData.totalImages || 0

  // 计算登录有效期百分比
  const getValidityPercent = () => {
    if (!loginStatus?.hoursUntilExpire) return 0
    const maxHours = 96 // 4天
    return Math.min(100, (loginStatus.hoursUntilExpire / maxHours) * 100)
  }

  if (checking) {
    return (
      <div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Spin size="large" />
      </div>
    )
  }

  return (
    <div style={{
      height: '100%',
      display: 'flex',
      flexDirection: 'column',
      gap: 12,
      overflow: 'hidden',
    }}>
      {/* 顶部欢迎区 */}
      <Card
        style={{
          background: 'linear-gradient(135deg, #07C160 0%, #06AE56 100%)',
          border: 'none',
          flex: '0 0 auto',
          position: 'relative',
          overflow: 'hidden',
        }}
        bodyStyle={{ padding: 20, position: 'relative', zIndex: 1 }}
      >
        {/* 音乐节奏条动效 */}
        <div className="wave-container">
          <div className="wave wave-1"></div>
          <div className="wave wave-2"></div>
          <div className="wave wave-3"></div>
          <div className="wave wave-4"></div>
          <div className="wave wave-5"></div>
          <div className="wave wave-6"></div>
          <div className="wave wave-7"></div>
          <div className="wave wave-8"></div>
        </div>

        <Row align="middle">
          <Col>
            <Space size={8} align="center">
              <div style={{
                width: 53,
                height: 53,
                borderRadius: 12,
                background: 'rgba(255, 255, 255, 0.2)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}>
                <WechatOutlined style={{ fontSize: 28, color: '#fff' }} />
              </div>
              <div style={{ textAlign: 'left' }}>
                <div style={{ fontSize: 20, fontWeight: 600, color: '#fff', marginBottom: 1 }}>
                  WeMedia Spider
                </div>
                <div style={{ fontSize: 13, color: 'rgba(255, 255, 255, 0.9)' }}>
                  微信公众号文章智能爬虫 · {dayjs().format('YYYY年MM月DD日')}
                </div>
              </div>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 主内容区 */}
      <Row gutter={12} style={{ flex: 1, minHeight: 0 }}>
        {/* 左侧：统计数据和功能 */}
        <Col span={16} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {/* 统计卡片 */}
          <Row gutter={12}>
            <Col span={6}>
              <Card
                style={{
                  background: 'linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%)',
                  border: '1px solid rgba(7, 193, 96, 0.3)',
                }}
                bodyStyle={{ padding: 16 }}
              >
                <Statistic
                  title={<span style={{ color: '#999', fontSize: 12 }}>总文章数</span>}
                  value={totalArticles}
                  suffix="篇"
                  valueStyle={{ color: '#07C160', fontSize: 24, fontWeight: 600 }}
                  prefix={<FileTextOutlined />}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card
                style={{
                  background: 'linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%)',
                  border: '1px solid rgba(24, 144, 255, 0.3)',
                }}
                bodyStyle={{ padding: 16 }}
              >
                <Statistic
                  title={<span style={{ color: '#999', fontSize: 12 }}>公众号数</span>}
                  value={accountCount}
                  suffix="个"
                  valueStyle={{ color: '#1890ff', fontSize: 24, fontWeight: 600 }}
                  prefix={<TeamOutlined />}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card
                style={{
                  background: 'linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%)',
                  border: '1px solid rgba(114, 46, 209, 0.3)',
                }}
                bodyStyle={{ padding: 16 }}
              >
                <Statistic
                  title={<span style={{ color: '#999', fontSize: 12 }}>图片下载</span>}
                  value={totalImages}
                  suffix="张"
                  valueStyle={{ color: '#722ed1', fontSize: 24, fontWeight: 600 }}
                  prefix={<PictureOutlined />}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card
                style={{
                  background: 'linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%)',
                  border: '1px solid rgba(250, 173, 20, 0.3)',
                }}
                bodyStyle={{ padding: 16 }}
              >
                <Statistic
                  title={<span style={{ color: '#999', fontSize: 12 }}>今日文章</span>}
                  value={todayArticles}
                  suffix="篇"
                  valueStyle={{ color: '#faad14', fontSize: 24, fontWeight: 600 }}
                  prefix={<FireOutlined />}
                />
              </Card>
            </Col>
          </Row>

          {/* 核心功能 */}
          <Card
            title={
              <Space>
                <ThunderboltOutlined style={{ color: '#07C160' }} />
                <span style={{ fontSize: 14, fontWeight: 600 }}>核心功能</span>
              </Space>
            }
            style={{
              flex: 1,
              background: 'rgba(255, 255, 255, 0.02)',
              border: '1px solid rgba(255, 255, 255, 0.08)',
            }}
            bodyStyle={{ padding: 16, height: 'calc(100% - 57px)' }}
          >
            <Row gutter={[12, 8]} style={{ height: '100%' }}>
              <Col span={24}>
                <div style={{
                  padding: '10px 12px',
                  background: 'linear-gradient(135deg, rgba(7, 193, 96, 0.1) 0%, rgba(7, 193, 96, 0.05) 100%)',
                  borderRadius: 6,
                  border: '1px solid rgba(7, 193, 96, 0.3)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: 16,
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <RocketOutlined style={{ fontSize: 18, color: '#07C160' }} />
                    <span style={{ fontSize: 13, fontWeight: 600, color: '#07C160' }}>高效批量爬取</span>
                  </div>
                  <div style={{ fontSize: 11, color: '#999', textAlign: 'right' }}>
                    多公众号并发爬取，智能频率控制，自动去重与增量更新
                  </div>
                </div>
              </Col>
              <Col span={24}>
                <div style={{
                  padding: '10px 12px',
                  background: 'linear-gradient(135deg, rgba(24, 144, 255, 0.1) 0%, rgba(24, 144, 255, 0.05) 100%)',
                  borderRadius: 6,
                  border: '1px solid rgba(24, 144, 255, 0.3)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: 16,
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <SafetyOutlined style={{ fontSize: 18, color: '#1890ff' }} />
                    <span style={{ fontSize: 13, fontWeight: 600, color: '#1890ff' }}>安全本地存储</span>
                  </div>
                  <div style={{ fontSize: 11, color: '#999', textAlign: 'right' }}>
                    AES-256加密存储凭证，4天有效期，支持导入导出备份
                  </div>
                </div>
              </Col>
              <Col span={24}>
                <div style={{
                  padding: '10px 12px',
                  background: 'linear-gradient(135deg, rgba(114, 46, 209, 0.1) 0%, rgba(114, 46, 209, 0.05) 100%)',
                  borderRadius: 6,
                  border: '1px solid rgba(114, 46, 209, 0.3)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: 16,
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <ThunderboltOutlined style={{ fontSize: 18, color: '#722ed1' }} />
                    <span style={{ fontSize: 13, fontWeight: 600, color: '#722ed1' }}>灵活筛选配置</span>
                  </div>
                  <div style={{ fontSize: 11, color: '#999', textAlign: 'right' }}>
                    日期范围、关键词过滤、并发线程数、请求间隔可自定义
                  </div>
                </div>
              </Col>
              <Col span={24}>
                <div style={{
                  padding: '10px 12px',
                  background: 'linear-gradient(135deg, rgba(250, 173, 20, 0.1) 0%, rgba(250, 173, 20, 0.05) 100%)',
                  borderRadius: 6,
                  border: '1px solid rgba(250, 173, 20, 0.3)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: 16,
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <TrophyOutlined style={{ fontSize: 18, color: '#faad14' }} />
                    <span style={{ fontSize: 13, fontWeight: 600, color: '#faad14' }}>多格式导出</span>
                  </div>
                  <div style={{ fontSize: 11, color: '#999', textAlign: 'right' }}>
                    CSV、JSON、Excel、Markdown格式，含完整元数据与正文内容
                  </div>
                </div>
              </Col>
              <Col span={24}>
                <div style={{
                  padding: '10px 12px',
                  background: 'linear-gradient(135deg, rgba(255, 77, 79, 0.1) 0%, rgba(255, 77, 79, 0.05) 100%)',
                  borderRadius: 6,
                  border: '1px solid rgba(255, 77, 79, 0.3)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: 16,
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <PictureOutlined style={{ fontSize: 18, color: '#ff4d4f' }} />
                    <span style={{ fontSize: 13, fontWeight: 600, color: '#ff4d4f' }}>批量图片下载</span>
                  </div>
                  <div style={{ fontSize: 11, color: '#999', textAlign: 'right' }}>
                    多线程并发下载，按公众号/文章分类，顺序编号保持一致
                  </div>
                </div>
              </Col>
              <Col span={24}>
                <div style={{
                  padding: '10px 12px',
                  background: 'linear-gradient(135deg, rgba(19, 194, 194, 0.1) 0%, rgba(19, 194, 194, 0.05) 100%)',
                  borderRadius: 6,
                  border: '1px solid rgba(19, 194, 194, 0.3)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: 16,
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <ClockCircleOutlined style={{ fontSize: 18, color: '#13c2c2' }} />
                    <span style={{ fontSize: 13, fontWeight: 600, color: '#13c2c2' }}>智能缓存机制</span>
                  </div>
                  <div style={{ fontSize: 11, color: '#999', textAlign: 'right' }}>
                    本地缓存已爬数据，避免重复请求，可配置过期时间策略
                  </div>
                </div>
              </Col>
            </Row>
          </Card>
        </Col>

        {/* 右侧：登录状态 */}
        <Col span={8}>
          <Card
            title={
              <Space>
                <LoginOutlined style={{ color: '#07C160' }} />
                <span style={{ fontSize: 14, fontWeight: 600 }}>登录状态</span>
              </Space>
            }
            style={{
              height: '100%',
              background: 'rgba(255, 255, 255, 0.02)',
              border: '1px solid rgba(255, 255, 255, 0.08)',
              display: 'flex',
              flexDirection: 'column',
            }}
            bodyStyle={{ padding: 16, flex: 1, display: 'flex', flexDirection: 'column' }}
          >
            {loginStatus?.isLoggedIn ? (
              <Space direction="vertical" size={12} style={{ width: '100%', flex: 1 }}>
                <div style={{
                  padding: 16,
                  background: 'rgba(7, 193, 96, 0.1)',
                  borderRadius: 8,
                  border: '1px solid rgba(7, 193, 96, 0.3)',
                  textAlign: 'center',
                }}>
                  <CheckCircleOutlined style={{ fontSize: 36, color: '#07C160', marginBottom: 8 }} />
                  <div style={{ fontSize: 15, fontWeight: 600, color: '#07C160', marginBottom: 4 }}>
                    登录成功
                  </div>
                  <div style={{ fontSize: 12, color: '#999' }}>
                    {loginStatus.message}
                  </div>
                </div>

                <div style={{
                  padding: 12,
                  background: '#1f1f1f',
                  borderRadius: 6,
                }}>
                  <div style={{ fontSize: 12, color: '#999', marginBottom: 8 }}>凭证有效期</div>
                  <Progress
                    percent={Math.round(getValidityPercent())}
                    strokeColor={{
                      '0%': '#07C160',
                      '100%': '#06AE56',
                    }}
                    trailColor="rgba(255, 255, 255, 0.1)"
                    format={() => (
                      <span style={{ color: '#07C160', fontSize: 14, fontWeight: 600 }}>
                        {loginStatus.hoursUntilExpire !== undefined
                          ? `${loginStatus.hoursUntilExpire.toFixed(1)}h`
                          : '-'
                        }
                      </span>
                    )}
                  />
                </div>

                <Divider style={{ margin: '8px 0', borderColor: 'rgba(255, 255, 255, 0.1)' }} />

                <div style={{
                  padding: 12,
                  background: '#1f1f1f',
                  borderRadius: 6,
                }}>
                  <Row gutter={12}>
                    <Col span={12}>
                      <div style={{ fontSize: 11, color: '#999', marginBottom: 4 }}>登录时间</div>
                      <div style={{ fontSize: 12, fontWeight: 500, color: '#fff' }}>
                        {loginStatus.loginTime
                          ? dayjs(loginStatus.loginTime).format('MM-DD HH:mm')
                          : '-'
                        }
                      </div>
                    </Col>
                    <Col span={12}>
                      <div style={{ fontSize: 11, color: '#999', marginBottom: 4 }}>过期时间</div>
                      <div style={{ fontSize: 12, fontWeight: 500, color: '#fff' }}>
                        {loginStatus.expireTime
                          ? dayjs(loginStatus.expireTime).format('MM-DD HH:mm')
                          : '-'
                        }
                      </div>
                    </Col>
                  </Row>
                </div>

                <div style={{ flex: 1 }} />

                <Button
                  icon={<ExportOutlined />}
                  onClick={handleExportCredentials}
                  loading={loading}
                  block
                  style={{ height: 36 }}
                >
                  导出凭证
                </Button>
                <Button
                  icon={<LogoutOutlined />}
                  onClick={handleLogout}
                  loading={loading}
                  block
                  danger
                  style={{ height: 36 }}
                >
                  退出登录
                </Button>
              </Space>
            ) : (
              <Space direction="vertical" size={12} style={{ width: '100%', flex: 1 }}>
                <div style={{
                  padding: 20,
                  background: 'rgba(250, 173, 20, 0.1)',
                  borderRadius: 8,
                  border: '1px solid rgba(250, 173, 20, 0.3)',
                  textAlign: 'center',
                }}>
                  <CloseCircleOutlined style={{ fontSize: 36, color: '#faad14', marginBottom: 8 }} />
                  <div style={{ fontSize: 15, fontWeight: 600, color: '#fff', marginBottom: 4 }}>
                    未登录
                  </div>
                  <div style={{ fontSize: 12, color: '#999', lineHeight: 1.6 }}>
                    请登录微信公众号平台后开始使用
                  </div>
                </div>

                <div style={{
                  padding: 16,
                  background: '#1f1f1f',
                  borderRadius: 6,
                }}>
                  <div style={{ fontSize: 13, fontWeight: 600, color: '#fff', marginBottom: 8 }}>
                    登录说明
                  </div>
                  <div style={{ fontSize: 12, color: '#999', lineHeight: 1.8 }}>
                    • 点击"立即登录"打开浏览器<br />
                    • 使用微信扫码登录公众号平台<br />
                    • 登录成功后凭证有效期4天<br />
                    • 支持导入导出凭证功能
                  </div>
                </div>

                <div style={{ flex: 1 }} />

                <Button
                  type="primary"
                  icon={<LoginOutlined />}
                  onClick={handleLogin}
                  loading={loading}
                  block
                  style={{ height: 44, fontSize: 15, fontWeight: 500 }}
                >
                  {loading ? '登录中...' : '立即登录'}
                </Button>
                <Button
                  icon={<ImportOutlined />}
                  onClick={handleImportCredentials}
                  loading={loading}
                  block
                  style={{ height: 36 }}
                >
                  导入凭证
                </Button>
              </Space>
            )}
          </Card>
        </Col>
      </Row>

      {/* 更新提示对话框 */}
      <Modal
        title={
          <Space>
            <GiftOutlined style={{ color: '#07C160' }} />
            <span>发现新版本</span>
          </Space>
        }
        open={showUpdateModal}
        onCancel={() => setShowUpdateModal(false)}
        footer={[
          <Button key="later" onClick={() => setShowUpdateModal(false)}>
            稍后更新
          </Button>,
          <Button
            key="download"
            type="primary"
            icon={<CloudDownloadOutlined />}
            onClick={() => {
              if (updateInfo?.updateUrl) {
                window.open(updateInfo.updateUrl, '_blank')
              }
              setShowUpdateModal(false)
            }}
          >
            立即下载
          </Button>,
        ]}
        width={600}
      >
        <div style={{ padding: '16px 0' }}>
          <Alert
            message={
              <Space>
                <span>当前版本: {updateInfo?.currentVersion}</span>
                <span>→</span>
                <span style={{ color: '#07C160', fontWeight: 500 }}>
                  最新版本: {updateInfo?.latestVersion}
                </span>
              </Space>
            }
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
          />

          {updateInfo?.releaseNotes && (
            <div>
              <div style={{ fontWeight: 500, marginBottom: 8 }}>更新内容：</div>
              <div
                style={{
                  background: '#1a1a1a',
                  padding: 12,
                  borderRadius: 6,
                  maxHeight: 300,
                  overflow: 'auto',
                  whiteSpace: 'pre-wrap',
                  fontSize: 13,
                  lineHeight: 1.6,
                }}
              >
                {updateInfo.releaseNotes}
              </div>
            </div>
          )}
        </div>
      </Modal>
    </div>
  )
}

export default HomePage
