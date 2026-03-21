import React, { useState, useEffect } from 'react'
import {
  Card,
  Button,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  Switch,
  Tag,
  Tooltip,
  Popconfirm,
  App,
  TimePicker,
  Empty,
} from 'antd'
import {
  PlusOutlined,
  PlayCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  ClockCircleOutlined,
  HistoryOutlined,
} from '@ant-design/icons'
import dayjs from 'dayjs'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import {
  ListScheduledTasks,
  CreateScheduledTask,
  UpdateScheduledTask,
  DeleteScheduledTask,
  RunScheduledTaskNow,
  GetTaskExecutionLogs,
} from '../../wailsjs/go/app/App'
import { models } from '../../wailsjs/go/models'

const { TextArea } = Input

// ============================================================
// Cron 工具函数
// ============================================================

type FrequencyType = 'daily' | 'weekly' | 'interval'

interface ScheduleFields {
  frequency: FrequencyType
  hour: number
  minute: number
  weekday: number
  intervalHours: number
}

/** 从可视化字段生成 cron 表达式（秒级，6位） */
const buildCron = (fields: ScheduleFields): string => {
  const { frequency, hour, minute, weekday, intervalHours } = fields
  switch (frequency) {
    case 'daily':
      return `0 ${minute} ${hour} * * *`
    case 'weekly':
      return `0 ${minute} ${hour} * * ${weekday}`
    case 'interval':
      return `0 0 */${intervalHours} * * *`
    default:
      return '0 0 2 * * *'
  }
}

/** 从 cron 表达式解析回可视化字段 */
const parseCron = (expr: string): ScheduleFields => {
  const parts = expr.split(/\s+/)
  // 6位: 秒 分 时 日 月 周
  if (parts.length !== 6) return { frequency: 'daily', hour: 2, minute: 0, weekday: 1, intervalHours: 6 }

  const [, min, hr, , , dow] = parts

  // 每隔N小时: 0 0 */N * * *
  if (hr.startsWith('*/')) {
    return { frequency: 'interval', hour: 0, minute: 0, weekday: 1, intervalHours: parseInt(hr.slice(2)) || 6 }
  }
  // 每周: dow 不是 *
  if (dow !== '*') {
    return { frequency: 'weekly', hour: parseInt(hr) || 0, minute: parseInt(min) || 0, weekday: parseInt(dow) || 1, intervalHours: 6 }
  }
  // 每天
  return { frequency: 'daily', hour: parseInt(hr) || 0, minute: parseInt(min) || 0, weekday: 1, intervalHours: 6 }
}

/** cron 表达式转中文描述 */
const cronToText = (expr: string): string => {
  const f = parseCron(expr)
  const pad = (n: number) => String(n).padStart(2, '0')
  const weekNames = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  switch (f.frequency) {
    case 'daily':
      return `每天 ${pad(f.hour)}:${pad(f.minute)}`
    case 'weekly':
      return `${weekNames[f.weekday] || '周' + f.weekday} ${pad(f.hour)}:${pad(f.minute)}`
    case 'interval':
      return `每 ${f.intervalHours} 小时`
    default:
      return expr
  }
}

// ============================================================
// TaskCard 组件
// ============================================================

interface TaskCardProps {
  task: models.ScheduledTask
  onEdit: (task: models.ScheduledTask) => void
  onDelete: (id: number) => void
  onToggle: (task: models.ScheduledTask) => void
  onRunNow: (id: number) => void
  onViewLogs: (id: number) => void
}

const TaskCard: React.FC<TaskCardProps> = ({ task, onEdit, onDelete, onToggle, onRunNow, onViewLogs }) => {
  const getStatusColor = () => {
    if (!task.enabled) return '#555'
    if (task.lastRunStatus === 'running') return '#1677ff'
    if (task.lastRunStatus === 'success') return '#07C160'
    if (task.lastRunStatus === 'failed') return '#ff4d4f'
    return '#faad14'
  }

  const getStatusText = () => {
    if (!task.enabled) return '已禁用'
    if (task.lastRunStatus === 'running') return '运行中'
    if (task.lastRunStatus === 'success') return '上次成功'
    if (task.lastRunStatus === 'failed') return '上次失败'
    return '待运行'
  }

  // 解析 scrapeConfig 获取公众号数量
  let accountCount = 0
  try {
    const cfg = JSON.parse(task.scrapeConfig || '{}')
    accountCount = cfg.accounts?.length || 0
  } catch { /* ignore */ }

  const accentColor = getStatusColor()

  return (
    <Card
      style={{
        background: 'rgba(255,255,255,0.04)',
        borderLeft: `3px solid ${accentColor}`,
        borderTop: '1px solid rgba(255,255,255,0.08)',
        borderRight: '1px solid rgba(255,255,255,0.08)',
        borderBottom: '1px solid rgba(255,255,255,0.08)',
      }}
      bodyStyle={{ padding: '10px 14px' }}
    >
      {/* 第一行：名称 + 操作按钮 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, minWidth: 0 }}>
          <span
            style={{
              display: 'inline-block', width: 6, height: 6, borderRadius: '50%',
              background: accentColor, flexShrink: 0,
            }}
          />
          <span style={{ fontSize: 13, fontWeight: 600, color: 'rgba(255,255,255,0.92)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {task.name}
          </span>
          <span style={{ fontSize: 11, color: accentColor, flexShrink: 0 }}>{getStatusText()}</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexShrink: 0, marginLeft: 8 }}>
          <Tooltip title={task.enabled ? '禁用' : '启用'}>
            <Switch size="small" checked={task.enabled} onChange={() => onToggle(task)} />
          </Tooltip>
          <Tooltip title="立即运行">
            <PlayCircleOutlined
              style={{ fontSize: 14, cursor: task.enabled ? 'pointer' : 'not-allowed', color: task.enabled ? '#07C160' : 'rgba(255,255,255,0.25)' }}
              onClick={() => task.enabled && onRunNow(task.ID)}
            />
          </Tooltip>
          <Tooltip title="日志">
            <HistoryOutlined
              style={{ fontSize: 14, cursor: 'pointer', color: 'rgba(255,255,255,0.5)' }}
              onClick={() => onViewLogs(task.ID)}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <EditOutlined
              style={{ fontSize: 14, cursor: 'pointer', color: 'rgba(255,255,255,0.5)' }}
              onClick={() => onEdit(task)}
            />
          </Tooltip>
          <Tooltip title="删除">
            <Popconfirm title="确认删除此任务？" onConfirm={() => onDelete(task.ID)} okText="删除" cancelText="取消" okType="danger">
              <DeleteOutlined style={{ fontSize: 14, cursor: 'pointer', color: 'rgba(255,77,79,0.7)' }} />
            </Popconfirm>
          </Tooltip>
        </div>
      </div>

      {/* 第二行：调度时间 + 公众号数 */}
      <div style={{ display: 'flex', gap: 16, fontSize: 12, color: 'rgba(255,255,255,0.55)', marginBottom: 6 }}>
        <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <ClockCircleOutlined style={{ fontSize: 11 }} />
          <span style={{ color: 'rgba(255,255,255,0.8)' }}>{cronToText(task.cronExpression)}</span>
        </span>
        {accountCount > 0 && (
          <span>{accountCount} 个公众号</span>
        )}
        <span>共 {task.totalRuns} 次 · {task.successRuns} 成功</span>
      </div>

      {/* 第三行：下次执行 */}
      {task.nextRunTime && (
        <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.35)' }}>
          下次: {new Date(task.nextRunTime).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })}
        </div>
      )}
    </Card>
  )
}

// ============================================================
// SchedulePage 主组件
// ============================================================

const SchedulePage: React.FC = () => {
  const { message: antMessage } = App.useApp()
  const [tasks, setTasks] = useState<models.ScheduledTask[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingTask, setEditingTask] = useState<models.ScheduledTask | null>(null)
  const [frequency, setFrequency] = useState<FrequencyType>('daily')
  const [logModalVisible, setLogModalVisible] = useState(false)
  const [logs, setLogs] = useState<any[]>([])
  const [form] = Form.useForm()

  const loadTasks = async () => {
    try {
      setLoading(true)
      const result = await ListScheduledTasks(false)
      setTasks(result || [])
    } catch (error) {
      console.error('加载任务列表失败:', error)
      antMessage.error('加载任务列表失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadTasks()
    const off = EventsOn('task:completed', (data: any) => {
      loadTasks()
      if (data?.status === 'success') {
        antMessage.success(`任务执行完成，获取 ${data.articles || 0} 篇文章`)
      } else if (data?.status === 'failed') {
        antMessage.error(`任务执行失败${data.errMsg ? '：' + data.errMsg : ''}`)
      }
    })
    return () => EventsOff('task:completed')
  }, [])

  // 打开创建/编辑对话框
  const handleOpenModal = (task?: models.ScheduledTask) => {
    if (task) {
      setEditingTask(task)
      const schedule = parseCron(task.cronExpression)
      setFrequency(schedule.frequency)
      let accounts = ''
      let recentDays = 30
      let maxPages = 10
      let requestInterval = 10
      let includeContent = false
      try {
        const cfg = JSON.parse(task.scrapeConfig || '{}')
        accounts = (cfg.accounts || []).join('\n')
        recentDays = cfg.recentDays || 30
        maxPages = cfg.maxPages || 10
        requestInterval = cfg.requestInterval || 10
        includeContent = cfg.includeContent || false
      } catch { /* ignore */ }
      form.setFieldsValue({
        name: task.name,
        description: task.description,
        frequency: schedule.frequency,
        time: dayjs().hour(schedule.hour).minute(schedule.minute),
        weekday: schedule.weekday,
        intervalHours: schedule.intervalHours,
        accounts,
        recentDays,
        maxPages,
        requestInterval,
        includeContent,
        enabled: task.enabled,
      })
    } else {
      setEditingTask(null)
      setFrequency('daily')
      form.resetFields()
    }
    setModalVisible(true)
  }

  // 提交创建/更新
  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()

      // 构建 cron 表达式
      const timeVal = values.time as dayjs.Dayjs
      const scheduleFields: ScheduleFields = {
        frequency: values.frequency,
        hour: timeVal ? timeVal.hour() : 2,
        minute: timeVal ? timeVal.minute() : 0,
        weekday: values.weekday || 1,
        intervalHours: values.intervalHours || 6,
      }
      const cronExpression = buildCron(scheduleFields)

      // 构建爬取配置
      const accounts = (values.accounts || '')
        .split('\n')
        .map((s: string) => s.trim())
        .filter((s: string) => s.length > 0)

      if (accounts.length === 0) {
        antMessage.warning('请输入至少一个公众号')
        return
      }

      const scrapeConfig = JSON.stringify({
        accounts,
        recentDays: values.recentDays || 30,
        startDate: '',
        endDate: '',
        maxPages: values.maxPages || 10,
        requestInterval: values.requestInterval || 10,
        includeContent: values.includeContent || false,
        keywordFilter: '',
        maxWorkers: 20,
      })

      const taskData = {
        name: values.name,
        description: values.description || '',
        cronExpression,
        enabled: values.enabled !== undefined ? values.enabled : true,
        scrapeConfig,
      }

      if (editingTask) {
        await UpdateScheduledTask(models.ScheduledTask.createFrom({ ...editingTask, ...taskData }))
        antMessage.success('任务更新成功')
      } else {
        await CreateScheduledTask(models.ScheduledTask.createFrom({
          ID: 0, ...taskData,
          lastRunStatus: '', lastRunError: '',
          totalRuns: 0, successRuns: 0, failedRuns: 0,
          createdAt: new Date().toISOString(), updatedAt: new Date().toISOString(),
        }))
        antMessage.success('任务创建成功')
      }

      setModalVisible(false)
      form.resetFields()
      setEditingTask(null)
      loadTasks()
    } catch (error) {
      console.error('保存任务失败:', error)
      antMessage.error('保存任务失败')
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await DeleteScheduledTask(id)
      antMessage.success('任务删除成功')
      loadTasks()
    } catch (error) {
      antMessage.error('删除任务失败')
    }
  }

  const handleToggle = async (task: models.ScheduledTask) => {
    try {
      await UpdateScheduledTask(models.ScheduledTask.createFrom({ ...task, enabled: !task.enabled }))
      antMessage.success(task.enabled ? '任务已禁用' : '任务已启用')
      loadTasks()
    } catch (error) {
      antMessage.error('更新任务状态失败')
    }
  }

  const handleRunNow = async (id: number) => {
    try {
      await RunScheduledTaskNow(id)
      antMessage.success('任务已开始执行')
    } catch (error) {
      antMessage.error('运行任务失败')
    }
  }

  const handleViewLogs = async (id: number) => {
    try {
      const result = await GetTaskExecutionLogs(id, 20)
      setLogs(result || [])
      setLogModalVisible(true)
    } catch (error) {
      antMessage.error('获取日志失败')
    }
  }

  const enabledTasks = tasks.filter(t => t.enabled).length

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden', padding: '12px 24px' }}>
      {/* 顶部操作栏 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 6 }}>
          <span style={{ fontSize: 22, fontWeight: 700, color: '#07C160', lineHeight: 1 }}>{enabledTasks}</span>
          <span style={{ fontSize: 13, color: 'rgba(255,255,255,0.45)' }}>/ {tasks.length} 个任务启用</span>
        </div>
        <Button type="primary" size="small" icon={<PlusOutlined />} onClick={() => handleOpenModal()}>创建任务</Button>
      </div>

      {/* 任务卡片网格 */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 12, alignContent: 'start', flex: 1, overflow: 'auto', paddingBottom: 12 }}>
        {tasks.length === 0 ? (
          <div style={{ gridColumn: '1 / -1', display: 'flex', justifyContent: 'center', alignItems: 'center', height: 300 }}>
            <Empty description="暂无定时任务，点击上方按钮创建" />
          </div>
        ) : tasks.map(task => (
          <TaskCard key={task.ID} task={task} onEdit={handleOpenModal} onDelete={handleDelete}
            onToggle={handleToggle} onRunNow={handleRunNow} onViewLogs={handleViewLogs} />
        ))}
      </div>

      {/* 创建/编辑任务对话框 */}
      <Modal
        title={editingTask ? '编辑任务' : '创建定时任务'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => { setModalVisible(false); form.resetFields(); setEditingTask(null) }}
        width={560}
        okText="保存"
        cancelText="取消"
      >
        <Form form={form} layout="vertical" initialValues={{
          enabled: true, frequency: 'daily', time: dayjs().hour(2).minute(0),
          weekday: 1, intervalHours: 6, recentDays: 30, maxPages: 10, requestInterval: 10, includeContent: false,
        }}>
          <Form.Item label="任务名称" name="name" rules={[{ required: true, message: '请输入任务名称' }]}>
            <Input placeholder="例如：每日采集任务" />
          </Form.Item>

          <Form.Item label="描述" name="description">
            <Input placeholder="任务描述（可选）" />
          </Form.Item>

          {/* 频率设置 */}
          <div style={{ display: 'flex', gap: 12, marginBottom: 0 }}>
            <Form.Item label="执行频率" name="frequency" style={{ flex: 1 }}>
              <Select onChange={(v: FrequencyType) => setFrequency(v)} options={[
                { label: '每天', value: 'daily' },
                { label: '每周', value: 'weekly' },
                { label: '每隔N小时', value: 'interval' },
              ]} />
            </Form.Item>

            {frequency !== 'interval' && (
              <Form.Item label="执行时间" name="time" style={{ flex: 1 }}>
                <TimePicker format="HH:mm" style={{ width: '100%' }} />
              </Form.Item>
            )}

            {frequency === 'weekly' && (
              <Form.Item label="星期" name="weekday" style={{ flex: 1 }}>
                <Select options={[
                  { label: '周一', value: 1 }, { label: '周二', value: 2 }, { label: '周三', value: 3 },
                  { label: '周四', value: 4 }, { label: '周五', value: 5 }, { label: '周六', value: 6 },
                  { label: '周日', value: 0 },
                ]} />
              </Form.Item>
            )}

            {frequency === 'interval' && (
              <Form.Item label="间隔小时" name="intervalHours" style={{ flex: 1 }}>
                <InputNumber min={1} max={24} style={{ width: '100%' }} />
              </Form.Item>
            )}
          </div>

          {/* 爬取配置 */}
          <Form.Item label="公众号列表" name="accounts" rules={[{ required: true, message: '请输入至少一个公众号' }]}>
            <TextArea rows={3} placeholder={'每行一个公众号名称\n例如：\n人民日报\n新华社'} />
          </Form.Item>

          <div style={{ display: 'flex', gap: 12 }}>
            <Form.Item label="采集天数" name="recentDays" tooltip="采集最近N天的文章" style={{ flex: 1 }}>
              <InputNumber min={1} max={365} style={{ width: '100%' }} addonAfter="天" />
            </Form.Item>
            <Form.Item label="最大页数" name="maxPages" style={{ flex: 1 }}>
              <InputNumber min={1} max={100} style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item label="请求间隔" name="requestInterval" style={{ flex: 1 }}>
              <InputNumber min={1} max={60} style={{ width: '100%' }} addonAfter="秒" />
            </Form.Item>
          </div>

          <div style={{ display: 'flex', gap: 24 }}>
            <Form.Item label="获取正文" name="includeContent" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item label="启用任务" name="enabled" valuePropName="checked">
              <Switch />
            </Form.Item>
          </div>
        </Form>
      </Modal>

      {/* 执行日志对话框 */}
      <Modal
        title="执行日志"
        open={logModalVisible}
        onCancel={() => setLogModalVisible(false)}
        footer={null}
        width={600}
      >
        {logs.length === 0 ? (
          <Empty description="暂无执行记录" />
        ) : (
          <div style={{ maxHeight: 400, overflow: 'auto' }}>
            {logs.map((log: any, idx: number) => (
              <div key={idx} style={{
                padding: '6px 12px', marginBottom: 6,
                background: 'rgba(255,255,255,0.03)', borderRadius: 6,
                fontSize: 12,
              }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                  <Tag color={log.status === 'success' ? 'success' : log.status === 'failed' ? 'error' : 'processing'} style={{ margin: 0 }}>
                    {log.status === 'success' ? '成功' : log.status === 'failed' ? '失败' : '运行中'}
                  </Tag>
                  <span style={{ color: 'rgba(255,255,255,0.65)' }}>
                    {new Date(log.startTime).toLocaleString('zh-CN')}
                    {log.duration ? ` · ${(log.duration / 1000).toFixed(1)}s` : ''}
                    {log.articlesCount ? ` · ${log.articlesCount} 篇文章` : ''}
                  </span>
                  <Tag style={{ margin: 0, fontSize: 11 }}>{log.triggerType === 'manual' ? '手动' : '定时'}</Tag>
                </div>
                {log.errorMessage && (
                  <div style={{ color: '#ff4d4f', marginTop: 4 }}>{log.errorMessage}</div>
                )}
              </div>
            ))}
          </div>
        )}
      </Modal>
    </div>
  )
}

export default SchedulePage
