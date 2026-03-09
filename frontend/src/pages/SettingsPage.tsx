import React, { useEffect, useState, useRef } from 'react'
import {
  Card,
  Form,
  InputNumber,
  Button,
  App,
  Row,
  Col,
} from 'antd'
import {
  ReloadOutlined,
  ThunderboltOutlined,
  ClockCircleOutlined,
  TeamOutlined,
  FileTextOutlined,
  DatabaseOutlined,
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
    } catch (error: any) {
      message.error('加载配置失败: ' + (error.message || '未知错误'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadConfig()
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
    </div>
  )
}

export default SettingsPage
