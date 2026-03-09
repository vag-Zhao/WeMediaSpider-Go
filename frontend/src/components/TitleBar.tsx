import React from 'react'
import {
  MinusOutlined,
  BorderOutlined,
  CloseOutlined,
  WechatOutlined,
} from '@ant-design/icons'
import { WindowMinimise, WindowToggleMaximise, Quit } from '../../wailsjs/runtime/runtime'
import './TitleBar.css'

const TitleBar: React.FC = () => {
  const handleMinimize = () => {
    WindowMinimise()
  }

  const handleMaximize = () => {
    WindowToggleMaximise()
  }

  const handleClose = () => {
    Quit()
  }

  return (
    <div className="title-bar" data-wails-drag>
      {/* 左侧 Logo */}
      <div className="title-bar-logo">
        <WechatOutlined style={{ fontSize: 18, color: '#07C160' }} />
      </div>

      {/* 中间标题 - 绝对定位居中 */}
      <div className="title-bar-title">
        WeMediaSpider - 微信公众号爬虫
      </div>

      {/* 右侧窗口控制按钮 */}
      <div className="title-bar-controls" data-wails-no-drag>
        <div className="title-bar-button minimize" onClick={handleMinimize}>
          <MinusOutlined style={{ fontSize: 12, color: '#fff' }} />
        </div>
        <div className="title-bar-button maximize" onClick={handleMaximize}>
          <BorderOutlined style={{ fontSize: 11, color: '#fff' }} />
        </div>
        <div className="title-bar-button close" onClick={handleClose}>
          <CloseOutlined style={{ fontSize: 12, color: '#fff' }} />
        </div>
      </div>
    </div>
  )
}

export default TitleBar
