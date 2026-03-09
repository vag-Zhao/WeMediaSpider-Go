import React, { useState } from 'react'
import { Layout as AntLayout, Menu } from 'antd'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  HomeOutlined,
  CloudDownloadOutlined,
  FileTextOutlined,
  SettingOutlined,
  WechatOutlined,
} from '@ant-design/icons'
import TitleBar from './TitleBar'
import PageTransition from './PageTransition'

const { Content, Sider } = AntLayout

interface LayoutProps {
  children: React.ReactNode
}

const Layout: React.FC<LayoutProps> = ({ children }) => {
  const navigate = useNavigate()
  const location = useLocation()
  const [collapsed, setCollapsed] = useState(true)

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
            }}
          />
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
