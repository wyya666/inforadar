import { useState } from 'react'
import { Alert, App, Button, Form, Input, Modal, Typography } from 'antd'
import { Link, useNavigate } from 'react-router-dom'
import type { User } from '../api'
import { useAuth } from '../auth/AuthProvider'
import { AppShell } from './AppShell'

interface PasswordValues { currentPassword: string; newPassword: string; confirmPassword: string }

export function SettingsPage() {
  const { user, request, setUser, endSession } = useAuth()
  const navigate = useNavigate()
  const { message } = App.useApp()
  const [profileError, setProfileError] = useState<string | null>(null)
  const [profileOpen, setProfileOpen] = useState(false)
  const [passwordError, setPasswordError] = useState<string | null>(null)
  const [passwordOpen, setPasswordOpen] = useState(false)
  const [savingProfile, setSavingProfile] = useState(false)
  const [savingPassword, setSavingPassword] = useState(false)

  const createdAt = user?.createdAt ? new Date(user.createdAt) : null
  const joinedDate = createdAt && !Number.isNaN(createdAt.getTime())
    ? new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' }).format(createdAt)
    : '—'

  async function saveProfile(values: { displayName: string }) {
    setProfileError(null)
    setSavingProfile(true)
    try {
      const updated = await request<User>('/users/me', { method: 'PATCH', body: JSON.stringify(values) })
      setUser(updated)
      setProfileOpen(false)
      void message.success('个人资料已保存')
    } catch (caught) {
      setProfileError(caught instanceof Error ? caught.message : '保存失败，请重试')
    } finally {
      setSavingProfile(false)
    }
  }

  async function changePassword(values: PasswordValues) {
    setPasswordError(null)
    setSavingPassword(true)
    try {
      await request<void>('/users/me/password', { method: 'POST', body: JSON.stringify({ currentPassword: values.currentPassword, newPassword: values.newPassword }) })
      endSession()
      navigate('/login', { replace: true, state: { notice: '密码已更新，请重新登录' } })
    } catch (caught) {
      setPasswordError(caught instanceof Error ? caught.message : '更新失败，请重试')
    } finally {
      setSavingPassword(false)
    }
  }

  function closePasswordModal() {
    if (savingPassword) return
    setPasswordOpen(false)
    setPasswordError(null)
  }

  function closeProfileModal() {
    if (savingProfile) return
    setProfileOpen(false)
    setProfileError(null)
  }

  return <AppShell><main className="settings-main">
    <div className="settings-heading"><Link to="/" className="back-link">← 返回主页</Link><h1>个人中心</h1><p>查看账号信息，管理个人资料和登录安全。</p></div>
    <div className="settings-grid">
      <section className="settings-card profile-card"><div className="section-kicker">PROFILE</div><h2>个人资料</h2>
        <div className="profile-identity"><span className="profile-avatar" aria-hidden="true">{user?.displayName?.slice(0, 1).toUpperCase() || '?'}</span><div><strong>{user?.displayName}</strong><span>当前登录账号</span></div></div>
        <dl className="profile-facts">
          <div><dt>登录邮箱</dt><dd>{user?.email}</dd></div>
          <div><dt>注册时间</dt><dd>{createdAt && !Number.isNaN(createdAt.getTime()) ? <time dateTime={user?.createdAt}>{joinedDate}</time> : joinedDate}</dd></div>
        </dl>
        <Button type="primary" onClick={() => { setProfileError(null); setProfileOpen(true) }}>编辑资料</Button>
      </section>
      <section className="settings-card security-card"><div className="section-kicker">SECURITY</div><h2>登录安全</h2><Typography.Paragraph type="secondary">定期更新密码，保护你的监控信息。修改成功后，所有设备都需要重新登录。</Typography.Paragraph>
        <Button type="primary" onClick={() => setPasswordOpen(true)}>修改密码</Button>
      </section>
    </div>
    <Modal title="编辑个人资料" open={profileOpen} onCancel={closeProfileModal} footer={null} destroyOnHidden mask={{ closable: false }} closable={!savingProfile} keyboard={!savingProfile} className="profile-modal">
      <p className="password-modal-intro">修改登录后显示的名称。</p>
      {profileError && <Alert type="error" title={profileError} showIcon className="form-alert" />}
      <Form<{ displayName: string }> layout="vertical" initialValues={{ displayName: user?.displayName ?? '' }} onFinish={saveProfile} requiredMark={false} preserve={false} clearOnDestroy>
        <Form.Item name="displayName" label="显示名称" rules={[{ required: true, whitespace: true, message: '请输入显示名称' }, { max: 256, message: '最多 256 个字符' }]}><Input placeholder="输入显示名称" maxLength={256} autoFocus /></Form.Item>
        <div className="settings-modal-actions"><Button onClick={closeProfileModal} disabled={savingProfile}>取消</Button><Button type="primary" htmlType="submit" loading={savingProfile}>保存资料</Button></div>
      </Form>
    </Modal>
    <Modal title="修改密码" open={passwordOpen} onCancel={closePasswordModal} footer={null} destroyOnHidden mask={{ closable: false }} closable={!savingPassword} keyboard={!savingPassword} className="password-modal">
      <p className="password-modal-intro">请输入当前密码，并设置新的登录密码。更新后所有设备都需要重新登录。</p>
      {passwordError && <Alert type="error" title={passwordError} showIcon className="form-alert" />}
      <Form<PasswordValues> layout="vertical" onFinish={changePassword} requiredMark={false} preserve={false} clearOnDestroy>
          <Form.Item name="currentPassword" label="当前密码" rules={[{ required: true, message: '请输入当前密码' }]}><Input.Password autoComplete="current-password" /></Form.Item>
          <Form.Item name="newPassword" label="新密码" rules={[{ required: true, message: '请输入新密码' }, { min: 8, message: '至少输入 8 个字符' }, { max: 128, message: '最多输入 128 个字符' }]}><Input.Password autoComplete="new-password" /></Form.Item>
          <Form.Item name="confirmPassword" label="确认新密码" dependencies={['newPassword']} rules={[{ required: true, message: '请再次输入新密码' }, ({ getFieldValue }) => ({ validator(_, value) { return value === getFieldValue('newPassword') ? Promise.resolve() : Promise.reject(new Error('两次输入的密码不一致')) } })]}><Input.Password autoComplete="new-password" /></Form.Item>
          <div className="settings-modal-actions"><Button onClick={closePasswordModal} disabled={savingPassword}>取消</Button><Button type="primary" htmlType="submit" loading={savingPassword}>更新密码</Button></div>
      </Form>
    </Modal>
  </main></AppShell>
}
