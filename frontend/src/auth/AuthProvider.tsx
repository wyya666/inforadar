import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { api, post } from '../api'
import type { LoginResponse, User } from '../api'

interface AuthContextValue {
  user: User | null
  ready: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  request: <T,>(path: string, init?: RequestInit) => Promise<T>
  setUser: (user: User) => void
  endSession: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)
let pendingRefresh: Promise<LoginResponse> | null = null
const refreshCookieLock = 'inforadar-refresh-cookie'

async function withRefreshCookieLock<T>(action: () => Promise<T>): Promise<T> {
  // Tabs share the HttpOnly cookie but not this module's pendingRefresh promise.
  // Web Locks serializes cookie rotation across tabs on the same origin.
  if (navigator.locks) return await navigator.locks.request(refreshCookieLock, action)
  return action()
}

function refreshOnce(): Promise<LoginResponse> {
  if (!pendingRefresh) {
    pendingRefresh = withRefreshCookieLock(() => post<LoginResponse>('/auth/refresh'))
      .finally(() => { pendingRefresh = null })
  }
  return pendingRefresh
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [token, setToken] = useState<string | null>(null)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    let active = true
    refreshOnce()
      .then((session) => {
        if (active) {
          setUser(session.user)
          setToken(session.accessToken)
        }
      })
      .catch(() => {})
      .finally(() => { if (active) setReady(true) })
    return () => { active = false }
  }, [])

  const endSession = useCallback(() => {
    setUser(null)
    setToken(null)
  }, [])

  const login = useCallback(async (email: string, password: string) => {
    const session = await post<LoginResponse>('/auth/login', { email, password })
    setUser(session.user)
    setToken(session.accessToken)
  }, [])

  const logout = useCallback(async () => {
    await withRefreshCookieLock(() => post<void>('/auth/logout'))
    endSession()
  }, [endSession])

  const request = useCallback(async <T,>(path: string, init: RequestInit = {}): Promise<T> => {
    if (!token) throw new Error('请先登录')
    try {
      return await api<T>(path, init, token)
    } catch (error) {
      if (!(error instanceof Error) || !('status' in error) || error.status !== 401) throw error
      try {
        const session = await refreshOnce()
        setUser(session.user)
        setToken(session.accessToken)
        return await api<T>(path, init, session.accessToken)
      } catch (refreshError) {
        endSession()
        throw refreshError
      }
    }
  }, [token, endSession])

  const value = useMemo(() => ({ user, ready, login, logout, request, setUser, endSession }),
    [user, ready, login, logout, request, endSession])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext)
  if (!context) throw new Error('AuthProvider is missing')
  return context
}
