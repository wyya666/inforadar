import { useState } from 'react'
import type { ReactNode } from 'react'
import { Button, Dropdown } from 'antd'
import { Link, NavLink, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthProvider'

export function AppShell({ children }: { children: ReactNode }) {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function doLogout() {
    setError(null)
    setLoading(true)
    try {
      await logout()
      navigate('/login', { replace: true })
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : '退出失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  return <div className="workspace">
    <header className="workspace-header">
      <Link to="/" className="brand brand-dark"><span className="brand-symbol"><span /></span><span>InfoRadar</span></Link>
      <nav aria-label="主导航"><NavLink to="/" end className={({ isActive }) => isActive && location.hash !== '#topics' ? 'active' : undefined}>信息雷达</NavLink><Link to="/#topics" className={location.pathname === '/' && location.hash === '#topics' ? 'active' : undefined}>监控主题</Link><NavLink to="/radar/settings">监控设置</NavLink><NavLink to="/settings">个人中心</NavLink></nav>
      <div className="workspace-account">
        <Dropdown menu={{ items: [
          { key: 'settings', label: <Link to="/settings">个人中心</Link> },
          { key: 'radar', label: <Link to="/radar/settings">监控设置</Link> },
          { key: 'logout', label: '退出登录' },
        ], onClick: ({ key }) => { if (key === 'logout') void doLogout() } }} placement="bottomRight">
          <Button type="text" className="account-trigger" loading={loading}><span className="avatar-letter">{user?.displayName?.slice(0, 1).toUpperCase()}</span><span className="account-name">{user?.displayName}</span></Button>
        </Dropdown>
      </div>
    </header>
    {error && <div className="workspace-error" role="alert">{error}</div>}
    {children}
  </div>
}
