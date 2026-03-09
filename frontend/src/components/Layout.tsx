import React, { useState } from 'react'
import { Layout as AntLayout, Menu, Button, App } from 'antd'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  HomeOutlined,
  CloudDownloadOutlined,
  FileTextOutlined,
  SettingOutlined,
  WechatOutlined,
  SyncOutlined,
} from '@ant-design/icons'
import TitleBar from './TitleBar'
import PageTransition from './PageTransition'
import { api } from '../services/api'

const { Content, Sider } = AntLayout

interface LayoutProps {
  children: React.ReactNode
}

const Layout: React.FC<LayoutProps> = ({ children }) => {
  const { message } = App.useApp()
  const navigate = useNavigate()
  const location = useLocation()
  const [collapsed, setCollapsed] = useState(true)
  const [checkingUpdate, setCheckingUpdate] = useState(false)

  const menuItems = [
    {
      key: '/',
      icon: <HomeOutlined />,
      label: '首页',
    },
    {
      key: '/scrape',
      icon: <CloudDownloadOutlined />,
      label: '爬取',
    },
    {
      key: '/results',
      icon: <FileTextOutlined />,
      label: '数据',
    },
    {
      key: '/settings',
      icon: <SettingOutlined />,
      label: '设置',
    },
  ]

  // 检查更新
  const handleCheckUpdate = async () => {
    try {
      setCheckingUpdate(true)
      const versionInfo = await api.checkForUpdates()
      if (versionInfo.hasUpdate) {
        message.success(`发现新版本 ${versionInfo.latestVersion}，请前往首页查看详情`)
      } else {
        message.success('当前已是最新版本')
      }
    } catch (error) {
      console.error('检查更新失败:', error)
      message.error('检查更新失败，请稍后重试')
    } finally {
      setCheckingUpdate(false)
    }
  }

  return (
    <AntLayout style={{ minHeight: '100vh', maxHeight: '100vh', overflow: 'hidden' }}>
      {/* 自定义标题栏 */}
      <TitleBar />

      <AntLayout style={{ height: 'calc(100vh - 40px)' }}>
        <Sider
          theme="dark"
          width={180}
          collapsedWidth={50}
          collapsible
          collapsed={collapsed}
          onCollapse={setCollapsed}
          style={{
            background: '#0d0d0d',
            overflow: 'hidden',
            display: 'flex',
            flexDirection: 'column',
          }}
          trigger={
            <div style={{ background: '#0d0d0d', borderTop: '1px solid #1f1f1f' }}>
              {collapsed ? '›' : '‹'}
            </div>
          }
        >
          <Menu
            theme="dark"
            mode="inline"
            selectedKeys={[location.pathname]}
            items={menuItems}
            onClick={({ key }) => navigate(key)}
            style={{
              background: '#0d0d0d',
              fontSize: 14,
              paddingTop: 8,
              flex: 1,
            }}
          />

          {/* 检查更新按钮 */}
          <div style={{
            padding: collapsed ? '8px 0' : '8px 16px',
            borderTop: '1px solid #1f1f1f',
            background: '#0d0d0d',
          }}>
            <Button
              type="text"
              icon={<SyncOutlined spin={checkingUpdate} />}
              onClick={handleCheckUpdate}
              loading={checkingUpdate}
              block
              style={{
                color: 'rgba(255, 255, 255, 0.65)',
                height: 40,
                display: 'flex',
                alignItems: 'center',
                justifyContent: collapsed ? 'center' : 'flex-start',
              }}
            >
              {!collapsed && '检查更新'}
            </Button>
          </div>
        </Sider>
        <AntLayout style={{ overflow: 'hidden' }}>
          <Content
            style={{
              padding: '12px',
              background: 'transparent',
              overflow: 'hidden',
              height: '100%',
            }}
          >
            <PageTransition>
              {children}
            </PageTransition>
          </Content>
        </AntLayout>
      </AntLayout>
    </AntLayout>
  )
}

export default Layout
