import { useState } from 'react'
import { Alert, Button, Form, Input } from 'antd'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthProvider'
import { AuthShell } from './AuthShell'

interface Values { email: string; password: string }

export function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const notice = (location.state as { notice?: string } | null)?.notice
  const destination = (location.state as { from?: string } | null)?.from || '/'

  async function submit(values: Values) {
    setError(null)
    setLoading(true)
    try {
      await login(values.email, values.password)
      navigate(destination, { replace: true })
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : '登录失败，请稍后再试')
    } finally {
      setLoading(false)
    }
  }

  return <AuthShell title="欢迎回来" description="登录账号，继续关注重要变化。">
    {notice && <Alert type="success" showIcon title={notice} className="form-alert" />}
    {error && <Alert type="error" showIcon title={error} className="form-alert" />}
    <Form<Values> layout="vertical" onFinish={submit} autoComplete="on" size="large" requiredMark={false}>
      <Form.Item name="email" label="邮箱" rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '请输入有效邮箱' }]}>
        <Input autoComplete="email" placeholder="name@example.com" />
      </Form.Item>
      <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}>
        <Input.Password autoComplete="current-password" placeholder="输入密码" />
      </Form.Item>
      <div className="form-utility"><Link to="/forgot-password">忘记密码？</Link></div>
      <Button type="primary" htmlType="submit" loading={loading} block className="primary-action">登录</Button>
    </Form>
    <p className="form-switch">还没有账号？<Link to="/register">立即注册</Link></p>
  </AuthShell>
}
