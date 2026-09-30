import { useState } from 'react'
import { Alert, Button, Form, Input } from 'antd'
import { Link, useNavigate } from 'react-router-dom'
import { post } from '../api'
import { AuthShell } from './AuthShell'

interface Values { email: string; password: string; confirmPassword: string }

export function RegisterPage() {
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  async function submit(values: Values) {
    setError(null)
    setLoading(true)
    try {
      await post('/auth/register', { email: values.email, password: values.password })
      navigate('/login', { replace: true, state: { notice: '注册成功，请登录' } })
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : '注册失败，请稍后再试')
    } finally {
      setLoading(false)
    }
  }

  return <AuthShell title="创建账号" description="从一个账号开始，持续关注你在意的事。">
    {error && <Alert type="error" showIcon title={error} className="form-alert" />}
    <Form<Values> layout="vertical" onFinish={submit} size="large" requiredMark={false}>
      <Form.Item name="email" label="邮箱" rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '请输入有效邮箱' }]}>
        <Input autoComplete="email" placeholder="name@example.com" />
      </Form.Item>
      <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }, { min: 8, message: '至少输入 8 个字符' }, { max: 128, message: '最多输入 128 个字符' }]}>
        <Input.Password autoComplete="new-password" placeholder="至少 8 个字符" />
      </Form.Item>
      <Form.Item name="confirmPassword" label="确认密码" dependencies={['password']} rules={[{ required: true, message: '请再次输入密码' }, ({ getFieldValue }) => ({ validator(_, value) { return value === getFieldValue('password') ? Promise.resolve() : Promise.reject(new Error('两次输入的密码不一致')) } })]}>
        <Input.Password autoComplete="new-password" placeholder="再次输入密码" />
      </Form.Item>
      <Button type="primary" htmlType="submit" loading={loading} block className="primary-action">创建账号</Button>
    </Form>
    <p className="form-switch">已有账号？<Link to="/login">返回登录</Link></p>
  </AuthShell>
}
