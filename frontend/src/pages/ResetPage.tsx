import { useState } from 'react'
import { Alert, Button, Form, Input } from 'antd'
import { Link, useSearchParams } from 'react-router-dom'
import { post } from '../api'
import { AuthShell } from './AuthShell'

interface Values { password: string; confirmPassword: string }

export function ResetPage() {
  const [params] = useSearchParams()
  const token = params.get('token')
  const [error, setError] = useState<string | null>(token ? null : '重置链接缺少令牌')
  const [done, setDone] = useState(false)
  const [loading, setLoading] = useState(false)

  async function submit(values: Values) {
    if (!token) return
    setError(null)
    setLoading(true)
    try {
      await post<void>('/auth/reset-password', { token, newPassword: values.password })
      setDone(true)
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : '重置失败，请稍后再试')
    } finally {
      setLoading(false)
    }
  }

  return <AuthShell title="设置新密码" description="密码更新后，所有设备都需要重新登录。">
    {done ? <div className="result-panel"><Alert type="success" showIcon title="密码已更新" /><Link to="/login"><Button type="primary" block className="primary-action">返回登录</Button></Link></div> : <>
      {error && <Alert type="error" showIcon title={error} className="form-alert" />}
      {token && <Form<Values> layout="vertical" onFinish={submit} size="large" requiredMark={false}>
        <Form.Item name="password" label="新密码" rules={[{ required: true, message: '请输入新密码' }, { min: 8, message: '至少输入 8 个字符' }, { max: 128, message: '最多输入 128 个字符' }]}><Input.Password autoComplete="new-password" placeholder="至少 8 个字符" /></Form.Item>
        <Form.Item name="confirmPassword" label="确认新密码" dependencies={['password']} rules={[{ required: true, message: '请再次输入新密码' }, ({ getFieldValue }) => ({ validator(_, value) { return value === getFieldValue('password') ? Promise.resolve() : Promise.reject(new Error('两次输入的密码不一致')) } })]}><Input.Password autoComplete="new-password" placeholder="再次输入新密码" /></Form.Item>
        <Button type="primary" htmlType="submit" loading={loading} block className="primary-action">更新密码</Button>
      </Form>}
      <p className="form-switch"><Link to="/forgot-password">重新申请重置链接</Link></p>
    </>}
  </AuthShell>
}
