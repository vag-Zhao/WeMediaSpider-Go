import React, { useEffect, useState, useRef } from 'react'
import {
  Card,
  Form,
  InputNumber,
  Button,
  App,
  Row,
  Col,
  Switch,
  Divider,
} from 'antd'
import {
  ReloadOutlined,
  ThunderboltOutlined,
  ClockCircleOutlined,
  TeamOutlined,
  FileTextOutlined,
  DatabaseOutlined,
  AppstoreOutlined,
  RocketOutlined,
  EyeInvisibleOutlined,
} from '@ant-design/icons'
import { useConfigStore } from '../stores/configStore'
import { useFormStore } from '../stores/formStore'
import { api } from '../services/api'
import type { Config } from '../types'
import './SettingsPage.css'

const SettingsPage: React.FC = () => {
  const { message } = App.useApp()
  const [form] = Form.useForm()
  const { config, setConfig } = useConfigStore()
  const formStore = useFormStore()
  const [loading, setLoading] = useState(false)
  const saveTimerRef = useRef<number | null>(null)

  // 托盘和自启动设置
  const [closeToTray, setCloseToTray] = useState(true)
  const [autostartEnabled, setAutostartEnabled] = useState(false)
  const [autostartSilent, setAutostartSilent] = useState(false)

  // 加载配置
  const loadConfig = async () => {
    try {
      setLoading(true)
      const cfg = await api.loadConfig()
      console.log('从后端加载的配置:', JSON.stringify(cfg))
      console.log('当前 formStore 状态:', JSON.stringify({
        maxPages: formStore.maxPages,
        requestInterval: formStore.requestInterval,
        maxWorkers: formStore.maxWorkers,
      }))
      setConfig(cfg)

      // 直接使用后端配置
      console.log('最终设置到表单的值:', JSON.stringify(cfg))
      form.setFieldsValue(cfg)

      // 同步更新 formStore
      formStore.setMaxPages(cfg.maxPages)
      formStore.setRequestInterval(cfg.requestInterval)
      formStore.setMaxWorkers(cfg.maxWorkers)

      // 加载托盘和自启动设置
      await loadSystemSettings()
    } catch (error: any) {
      message.error('加载配置失败: ' + (error.message || '未知错误'))
    } finally {
      setLoading(false)
    }
  }

  // 加载系统设置
  const loadSystemSettings = async () => {
    try {
      const { GetCloseToTray, IsAutostartEnabled, IsAutostartSilent } = await import('../../wailsjs/go/app/App')

      const closeToTrayValue = await GetCloseToTray()
      const autostartValue = await IsAutostartEnabled()
      const silentValue = await IsAutostartSilent()

      console.log('加载系统设置:', {
        closeToTray: closeToTrayValue,
        autostart: autostartValue,
        silent: silentValue
      })

      setCloseToTray(closeToTrayValue)
      setAutostartEnabled(autostartValue)
      setAutostartSilent(silentValue)
    } catch (error) {
      console.error('加载系统设置失败:', error)
    }
  }

  // 处理关闭到托盘切换
  const handleCloseToTrayChange = async (checked: boolean) => {
    try {
      const { SetCloseToTray, SetRememberChoice } = await import('../../wailsjs/go/app/App')
      await SetCloseToTray(checked)
      // 只有当用户打开"关闭到托盘"时，才自动记住选择
      // 如果用户关闭该功能，则清除记住选择，下次会弹窗询问
      if (checked) {
        await SetRememberChoice(true)
      } else {
        await SetRememberChoice(false)
      }
      setCloseToTray(checked)
      message.success(checked ? '已启用关闭到托盘' : '已禁用关闭到托盘')
    } catch (error: any) {
      message.error('设置失败: ' + (error.message || '未知错误'))
    }
  }

  // 处理自启动切换
  const handleAutostartChange = async (checked: boolean) => {
    try {
      const { SetAutostart } = await import('../../wailsjs/go/app/App')
      await SetAutostart(checked, autostartSilent)
      setAutostartEnabled(checked)
      message.success(checked ? '已启用开机自启动' : '已禁用开机自启动')
    } catch (error: any) {
      message.error('设置失败: ' + (error.message || '未知错误'))
    }
  }

  // 处理静默启动切换
  const handleSilentChange = async (checked: boolean) => {
    try {
      const { SetAutostart } = await import('../../wailsjs/go/app/App')
      // 如果自启动已启用，需要重新设置
      if (autostartEnabled) {
        await SetAutostart(true, checked)
      }
      setAutostartSilent(checked)
      message.success(checked ? '已启用静默启动' : '已禁用静默启动')
    } catch (error: any) {
      message.error('设置失败: ' + (error.message || '未知错误'))
    }
  }

  useEffect(() => {
    loadConfig()

    // 监听系统配置变化事件
    import('../../wailsjs/runtime/runtime').then(({ EventsOn, EventsOff }) => {
      const handleSystemConfigChanged = (data: any) => {
        console.log('系统配置已更新:', data)
        if (data.closeToTray !== undefined) {
          setCloseToTray(data.closeToTray)
        }
      }

      EventsOn('system-config-changed', handleSystemConfigChanged)

      // 清理函数
      return () => {
        EventsOff('system-config-changed')
      }
    })
  }, [])

  // 自动保存配置（延迟100ms）
  const autoSave = async (values: any) => {
    if (saveTimerRef.current) {
      clearTimeout(saveTimerRef.current)
    }

    saveTimerRef.current = window.setTimeout(async () => {
      try {
        console.log('保存配置:', JSON.stringify(values))
        await api.saveConfig(values as Config)
        setConfig(values as Config)

        // 同步更新formStore
        formStore.setMaxPages(values.maxPages)
        formStore.setRequestInterval(values.requestInterval)
        formStore.setMaxWorkers(values.maxWorkers)
        console.log('formStore 已更新:', JSON.stringify({
          maxPages: values.maxPages,
          requestInterval: values.requestInterval,
          maxWorkers: values.maxWorkers,
        }))
      } catch (error: any) {
        console.error('自动保存失败:', error)
      }
    }, 100)
  }

  // 表单值变化时自动保存
  const handleValuesChange = (_: any, allValues: any) => {
    console.log('表单值变化:', allValues)
    autoSave(allValues)
  }

  // 恢复默认配置
  const handleReset = async () => {
    try {
      setLoading(true)
      const defaultConfig = await api.getDefaultConfig()

      // 保存到后端
      await api.saveConfig(defaultConfig)

      // 更新全局状态
      setConfig(defaultConfig)

      // 更新表单
      form.setFieldsValue(defaultConfig)

      // 同步更新 formStore
      formStore.setMaxPages(defaultConfig.maxPages)
      formStore.setRequestInterval(defaultConfig.requestInterval)
      formStore.setMaxWorkers(defaultConfig.maxWorkers)

      message.success('已恢复默认配置')
    } catch (error: any) {
      message.error('恢复失败: ' + (error.message || '未知错误'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{
      height: '100%',
      background: '#1a1a1a',
      padding: '24px',
      overflow: 'auto'
    }}>
      {/* 爬取配置 */}
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <ThunderboltOutlined style={{ color: '#07C160', fontSize: 18 }} />
            <span style={{ fontSize: 16, fontWeight: 600 }}>爬取配置</span>
          </div>
        }
        extra={
          <Button
            icon={<ReloadOutlined />}
            onClick={handleReset}
            loading={loading}
            size="small"
          >
            恢复默认
          </Button>
        }
        style={{
          background: 'rgba(255, 255, 255, 0.02)',
          border: '1px solid rgba(255, 255, 255, 0.08)'
        }}
      >
        <Form
          form={form}
          layout="vertical"
          onValuesChange={handleValuesChange}
          initialValues={{
            maxPages: 10,
            requestInterval: 10,
            maxWorkers: 5,
            cacheExpireHours: 96,
          }}
        >
          <Row gutter={[16, 0]} wrap={false} style={{ minWidth: 0 }}>
            {/* 最大页数 */}
            <Col flex="1" style={{ minWidth: 0 }}>
              <Form.Item
                label={
                  <span style={{ color: '#fff', fontSize: 13, display: 'flex', alignItems: 'center', gap: '6px', whiteSpace: 'nowrap' }}>
                    <FileTextOutlined style={{ color: '#07C160' }} />
                    最大页数
                  </span>
                }
                name="maxPages"
                rules={[{ required: true }]}
              >
                <InputNumber
                  min={1}
                  max={100}
                  style={{ width: '100%' }}
                  suffix="页"
                  className="centered-input-number"
                />
              </Form.Item>
            </Col>

            {/* 请求间隔 */}
            <Col flex="1" style={{ minWidth: 0 }}>
              <Form.Item
                label={
                  <span style={{ color: '#fff', fontSize: 13, display: 'flex', alignItems: 'center', gap: '6px', whiteSpace: 'nowrap' }}>
                    <ClockCircleOutlined style={{ color: '#07C160' }} />
                    请求间隔
                  </span>
                }
                name="requestInterval"
                rules={[{ required: true }]}
              >
                <InputNumber
                  min={1}
                  max={60}
                  style={{ width: '100%' }}
                  suffix="秒"
                  className="centered-input-number"
                />
              </Form.Item>
            </Col>

            {/* 并发数 */}
            <Col flex="1" style={{ minWidth: 0 }}>
              <Form.Item
                label={
                  <span style={{ color: '#fff', fontSize: 13, display: 'flex', alignItems: 'center', gap: '6px', whiteSpace: 'nowrap' }}>
                    <TeamOutlined style={{ color: '#07C160' }} />
                    并发数
                  </span>
                }
                name="maxWorkers"
                rules={[{ required: true }]}
              >
                <InputNumber
                  min={1}
                  max={10}
                  style={{ width: '100%' }}
                  suffix="个"
                  className="centered-input-number"
                />
              </Form.Item>
            </Col>

            {/* 缓存过期 */}
            <Col flex="1" style={{ minWidth: 0 }}>
              <Form.Item
                label={
                  <span style={{ color: '#fff', fontSize: 13, display: 'flex', alignItems: 'center', gap: '6px', whiteSpace: 'nowrap' }}>
                    <DatabaseOutlined style={{ color: '#07C160' }} />
                    缓存过期
                  </span>
                }
                name="cacheExpireHours"
                rules={[{ required: true }]}
              >
                <InputNumber
                  min={1}
                  max={720}
                  style={{ width: '100%' }}
                  suffix="小时"
                  className="centered-input-number"
                />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Card>

      {/* 系统设置 */}
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <AppstoreOutlined style={{ color: '#07C160', fontSize: 18 }} />
            <span style={{ fontSize: 16, fontWeight: 600 }}>系统设置</span>
          </div>
        }
        style={{
          background: 'rgba(255, 255, 255, 0.02)',
          border: '1px solid rgba(255, 255, 255, 0.08)',
          marginTop: 16,
        }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          {/* 关闭到托盘 */}
          <div style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '12px 16px',
            background: 'rgba(255, 255, 255, 0.02)',
            borderRadius: 6,
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, flex: 1 }}>
              <EyeInvisibleOutlined style={{ color: '#07C160', fontSize: 18, flexShrink: 0 }} />
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flex: 1 }}>
                <div style={{ color: '#fff', fontSize: 14, fontWeight: 500 }}>关闭到托盘</div>
                <div style={{ color: 'rgba(255, 255, 255, 0.45)', fontSize: 12, marginLeft: 16 }}>
                  点击关闭按钮时最小化到系统托盘，而不是退出程序
                </div>
              </div>
            </div>
            <Switch
              checked={closeToTray}
              onChange={handleCloseToTrayChange}
              style={{ marginLeft: 16, flexShrink: 0 }}
            />
          </div>

          <Divider style={{ margin: 0, borderColor: 'rgba(255, 255, 255, 0.08)' }} />

          {/* 开机自启动 */}
          <div style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '12px 16px',
            background: 'rgba(255, 255, 255, 0.02)',
            borderRadius: 6,
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, flex: 1 }}>
              <RocketOutlined style={{ color: '#07C160', fontSize: 18, flexShrink: 0 }} />
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flex: 1 }}>
                <div style={{ color: '#fff', fontSize: 14, fontWeight: 500 }}>开机自启动</div>
                <div style={{ color: 'rgba(255, 255, 255, 0.45)', fontSize: 12, marginLeft: 16 }}>
                  Windows 启动时自动运行本程序
                </div>
              </div>
            </div>
            <Switch
              checked={autostartEnabled}
              onChange={handleAutostartChange}
              style={{ marginLeft: 16, flexShrink: 0 }}
            />
          </div>

          {/* 静默启动 */}
          {autostartEnabled && (
            <div style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              padding: '12px 16px 12px 46px',
              background: 'rgba(255, 255, 255, 0.02)',
              borderRadius: 6,
              marginLeft: 20,
            }}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flex: 1 }}>
                <div style={{ color: '#fff', fontSize: 14, fontWeight: 500 }}>静默启动</div>
                <div style={{ color: 'rgba(255, 255, 255, 0.45)', fontSize: 12, marginLeft: 16 }}>
                  启动时直接最小化到托盘，不显示主窗口
                </div>
              </div>
              <Switch
                checked={autostartSilent}
                onChange={handleSilentChange}
                style={{ marginLeft: 16, flexShrink: 0 }}
              />
            </div>
          )}

          <Divider style={{ margin: 0, borderColor: 'rgba(255, 255, 255, 0.08)' }} />

          {/* 重置关闭选择 */}
          <div style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '12px 16px',
            background: 'rgba(255, 255, 255, 0.02)',
            borderRadius: 6,
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, flex: 1 }}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flex: 1 }}>
                <div style={{ color: '#fff', fontSize: 14, fontWeight: 500 }}>重置关闭选择</div>
                <div style={{ color: 'rgba(255, 255, 255, 0.45)', fontSize: 12, marginLeft: 16 }}>
                  清除"记住我的选择"设置，下次关闭时重新询问
                </div>
              </div>
            </div>
            <Button
              size="small"
              onClick={async () => {
                try {
                  const { SetRememberChoice } = await import('../../wailsjs/go/app/App')
                  await SetRememberChoice(false)
                  message.success('已重置关闭选择')
                } catch (error: any) {
                  message.error('重置失败: ' + (error.message || '未知错误'))
                }
              }}
              style={{ marginLeft: 16, flexShrink: 0 }}
            >
              重置
            </Button>
          </div>
        </div>
      </Card>
    </div>
  )
}

export default SettingsPage
