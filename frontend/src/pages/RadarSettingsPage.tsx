import { useEffect, useState } from 'react'
import { Alert, Button, Form, Input, Popconfirm, Select, Skeleton, Tag, message } from 'antd'
import { Link } from 'react-router-dom'
import { useAuth } from '../auth/AuthProvider'
import type { Provider, RadarSettings, SearchProvider } from '../radar'
import { defaultModels, providerNames } from '../radar'
import { AppShell } from './AppShell'

interface Values { modelProvider: Provider; modelId: string; searchProvider: SearchProvider; intervalHours: 1 | 6 | 12 | 24 }

export function RadarSettingsPage() {
  const { request } = useAuth()
  const [messageApi, contextHolder] = message.useMessage()
  const [form] = Form.useForm<Values>()
  const [settings, setSettings] = useState<RadarSettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [keyInputs, setKeyInputs] = useState<Record<Provider, string>>({ deepseek: '', zhipu: '', bailian: '' })
  const [keyBusy, setKeyBusy] = useState<Provider | null>(null)

  async function reload() {
    const next = await request<RadarSettings>('/radar/settings')
    setSettings(next)
  }

  useEffect(() => { reload().catch((caught) => setError(caught instanceof Error ? caught.message : '加载失败')).finally(() => setLoading(false)) }, [])

  useEffect(() => {
    if (!loading && settings) form.setFieldsValue({ modelProvider: settings.modelProvider, modelId: settings.modelId, searchProvider: settings.searchProvider, intervalHours: settings.intervalHours })
  }, [form, loading, settings])

  async function saveSettings(values: Values) {
    setSaving(true); setError(null)
    try {
      const next = await request<RadarSettings>('/radar/settings', { method: 'PUT', body: JSON.stringify(values) })
      setSettings(next); messageApi.success('监控设置已保存')
    } catch (caught) { setError(caught instanceof Error ? caught.message : '保存失败') }
    finally { setSaving(false) }
  }

  async function saveKey(provider: Provider) {
    const apiKey = keyInputs[provider].trim()
    if (apiKey.length < 8) { messageApi.error('请输入有效的 API Key'); return }
    setKeyBusy(provider)
    try {
      await request(`/radar/keys/${provider}`, { method: 'PUT', body: JSON.stringify({ apiKey }) })
      setKeyInputs((current) => ({ ...current, [provider]: '' })); await reload(); messageApi.success(`${providerNames[provider]}密钥已保存`)
    } catch (caught) { messageApi.error(caught instanceof Error ? caught.message : '保存密钥失败') }
    finally { setKeyBusy(null) }
  }

  async function deleteKey(provider: Provider) {
    setKeyBusy(provider)
    try {
      await request(`/radar/keys/${provider}`, { method: 'DELETE' })
      await reload(); messageApi.success(`${providerNames[provider]}密钥已删除`)
    } catch (caught) { messageApi.error(caught instanceof Error ? caught.message : '删除密钥失败') }
    finally { setKeyBusy(null) }
  }

  return <AppShell>{contextHolder}<main className="radar-settings-main">
    <div className="settings-heading"><Link to="/" className="back-link">← 返回信息雷达</Link><h1>监控设置</h1><p>选择分析模型、搜索服务和扫描间隔。设置对账号下的所有主题生效。</p></div>
    {error && <Alert type="error" showIcon title={error} className="radar-alert" />}
    {loading ? <Skeleton active paragraph={{ rows: 8 }} /> : <div className="radar-settings-grid">
      <section className="settings-card"><div className="section-kicker">SCANNING</div><h2>扫描方式</h2>
        <Form<Values> form={form} layout="vertical" onFinish={saveSettings} requiredMark={false} onValuesChange={(changed) => {
          if (changed.modelProvider) form.setFieldValue('modelId', defaultModels[changed.modelProvider as Provider])
        }}>
          <Form.Item name="modelProvider" label="分析模型提供商" rules={[{ required: true, message: '请选择提供商' }]}><Select options={Object.entries(providerNames).map(([value, label]) => ({ value, label }))} /></Form.Item>
          <Form.Item name="modelId" label="模型 ID" extra="预填推荐值，也可以填写该平台支持的其他模型 ID。" rules={[{ required: true, whitespace: true, message: '请输入模型 ID' }, { max: 128, message: '最多 128 个字符' }]}><Input maxLength={128} /></Form.Item>
          <Form.Item name="searchProvider" label="网页搜索服务" rules={[{ required: true, message: '请选择搜索服务' }]}><Select options={[{ value: 'zhipu', label: '智谱 Web Search' }, { value: 'bailian', label: '阿里百炼联网搜索' }]} /></Form.Item>
          <Form.Item name="intervalHours" label="每个主题的扫描间隔" rules={[{ required: true, message: '请选择扫描间隔' }]}><Select options={[1, 6, 12, 24].map((value) => ({ value, label: `每 ${value} 小时` }))} /></Form.Item>
          <Button type="primary" htmlType="submit" loading={saving}>保存监控设置</Button>
        </Form>
        <p className="settings-note">每个启用的主题都会按此间隔扫描。更频繁的扫描或更多主题会增加你在服务商处的调用费用。</p>
      </section>
      <section className="settings-card"><div className="section-kicker">CREDENTIALS</div><h2>平台 API Key</h2>
        <p className="key-intro">密钥加密保存在服务端，保存后不再显示原文。模型和搜索选同一平台时共用一份密钥。</p>
        {(['deepseek', 'zhipu', 'bailian'] as Provider[]).map((provider) => <div className="key-row" key={provider}>
          <div className="key-row-head"><strong>{providerNames[provider]}</strong><Tag color={settings?.configuredKeys[provider] ? 'success' : 'default'}>{settings?.configuredKeys[provider] ? '已配置' : '未配置'}</Tag></div>
          <div className="key-row-input"><Input.Password aria-label={`${providerNames[provider]} API Key`} value={keyInputs[provider]} onChange={(event) => setKeyInputs((current) => ({ ...current, [provider]: event.target.value }))} placeholder="粘贴常规平台 API Key" autoComplete="off" /><Button onClick={() => saveKey(provider)} loading={keyBusy === provider}>保存</Button></div>
          {settings?.configuredKeys[provider] && <Popconfirm title={`删除 ${providerNames[provider]} 密钥？`} description="使用该平台的监控主题会停止扫描。" onConfirm={() => deleteKey(provider)} okText="删除" cancelText="取消"><Button type="link" danger size="small" className="key-delete">删除密钥</Button></Popconfirm>}
        </div>)}
      </section>
    </div>}
  </main></AppShell>
}
