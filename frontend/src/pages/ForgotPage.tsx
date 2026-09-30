import { useState } from 'react'
import { Alert, Button, Form, Input } from 'antd'
import { Link } from 'react-router-dom'
import { post } from '../api'
import { AuthShell } from './AuthShell'

export function ForgotPage() {
  const [error, setError] = useState<string | null>(null)
  const [sent, setSent] = useState(false)
  const [loading, setLoading] = useState(false)

  async function submit(values: { email: string }) {
    setError(null)
    setLoading(true)
    try {
      await post('/auth/forgot-password', values)
      setSent(true)
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : '请求失败，请稍后再试')
    } finally {
      setLoading(false)
    }
  }

  return <AuthShell title="找回密码" description="输入注册邮箱，我们会发送密码重置链接。">
    {sent ? <div className="result-panel"><Alert type="success" showIcon title="如果该邮箱已注册，我们会发送重置链接" /><Link to="/login">返回登录 →</Link></div> : <>
      {error && <Alert type="error" showIcon title={error} className="form-alert" />}
      <Form layout="vertical" onFinish={submit} size="large" requiredMark={false}>
        <Form.Item name="email" label="邮箱" rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '请输入有效邮箱' }]}><Input autoComplete="email" placeholder="name@example.com" /></Form.Item>
        <Button type="primary" htmlType="submit" loading={loading} block className="primary-action">发送重置链接</Button>
      </Form>
      <p className="form-switch"><Link to="/login">返回登录</Link></p>
    </>}
  </AuthShell>
}
