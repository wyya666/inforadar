import { BrowserRouter, Navigate, Outlet, Route, Routes, useLocation } from 'react-router-dom'
import { Spin } from 'antd'
import { AuthProvider, useAuth } from './auth/AuthProvider'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'
import { ForgotPage } from './pages/ForgotPage'
import { ResetPage } from './pages/ResetPage'
import { HomePage } from './pages/HomePage'
import { SettingsPage } from './pages/SettingsPage'
import { RadarSettingsPage } from './pages/RadarSettingsPage'

function LoadingScreen() {
  return <div className="loading-screen"><Spin size="large" description="正在恢复登录状态" /></div>
}

function ProtectedRoute() {
  const { user, ready } = useAuth()
  const location = useLocation()
  if (!ready) return <LoadingScreen />
  return user ? <Outlet /> : <Navigate to="/login" replace state={{ from: location.pathname }} />
}

function GuestRoute() {
  const { user, ready } = useAuth()
  if (!ready) return <LoadingScreen />
  return user ? <Navigate to="/" replace /> : <Outlet />
}

export default function App() {
  return <BrowserRouter><AuthProvider><Routes>
    <Route element={<GuestRoute />}>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
    </Route>
    <Route path="/forgot-password" element={<ForgotPage />} />
    <Route path="/reset-password" element={<ResetPage />} />
    <Route element={<ProtectedRoute />}>
      <Route path="/" element={<HomePage />} />
      <Route path="/settings" element={<SettingsPage />} />
      <Route path="/radar/settings" element={<RadarSettingsPage />} />
    </Route>
    <Route path="*" element={<Navigate to="/" replace />} />
  </Routes></AuthProvider></BrowserRouter>
}
