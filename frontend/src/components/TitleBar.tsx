import React from 'react'
import {
  MinusOutlined,
  BorderOutlined,
  CloseOutlined,
  WechatOutlined,
} from '@ant-design/icons'
import { Checkbox, App, Button } from 'antd'
import { WindowMinimise, WindowToggleMaximise, Quit } from '../../wailsjs/runtime/runtime'
import './TitleBar.css'

const TitleBar: React.FC = () => {
  const { modal } = App.useApp()

  const handleMinimize = () => {
    WindowMinimise()
  }

  const handleMaximize = () => {
    WindowToggleMaximise()
  }

  const handleClose = async () => {
    // 检查是否已经记住选择
    const { GetCloseToTray, GetRememberChoice, HideToTray, SetCloseToTray, SetRememberChoice, ForceQuit } = await import('../../wailsjs/go/app/App')

    const remembered = await GetRememberChoice()
    const closeToTray = await GetCloseToTray()

    console.log('Close button clicked - remembered:', remembered, 'closeToTray:', closeToTray)

    if (remembered) {
      // 已记住选择
      console.log('Using remembered choice')
      if (closeToTray) {
        console.log('Hiding to tray')
        await HideToTray()
      } else {
        console.log('Force quitting')
        await ForceQuit()
      }
      return
    }

    // 使用局部变量来跟踪复选框状态
    let shouldRemember = false

    // 显示确认对话框
    console.log('Showing close confirmation dialog')
    const instance = modal.confirm({
      title: '关闭确认',
      content: (
        <div>
          <p>您想要最小化到托盘还是退出程序？</p>
          <Checkbox
            defaultChecked={false}
            onChange={(e) => {
              shouldRemember = e.target.checked
            }}
            style={{ marginTop: 12 }}
          >
            记住我的选择，下次不再提示
          </Checkbox>
        </div>
      ),
      maskClosable: true, // 允许点击遮罩层关闭
      keyboard: true,     // 允许按 ESC 键关闭
      closable: true,     // 显示关闭按钮
      footer: (_, { OkBtn, CancelBtn }) => (
        <>
          <Button
            onClick={async () => {
              // 最小化到托盘
              if (shouldRemember) {
                await SetRememberChoice(true)
                await SetCloseToTray(true)
              }
              await HideToTray()
              instance.destroy()
            }}
            type="primary"
          >
            最小化到托盘
          </Button>
          <Button
            onClick={async () => {
              // 退出程序
              if (shouldRemember) {
                await SetRememberChoice(true)
                await SetCloseToTray(false)
              }
              await ForceQuit()
            }}
          >
            退出程序
          </Button>
        </>
      ),
    })
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