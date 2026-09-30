import { ApiBasePath, ApiErrorCode } from './constants'

export interface User {
  id: string
  email: string
  displayName: string
  createdAt: string
}

export interface LoginResponse {
  accessToken: string
  expiresIn: number
  user: User
}

interface ApiResponse<T> {
  code?: number
  data?: T
  message?: string
}

export class ApiError extends Error {
  constructor(public status: number, public code: number, message: string) {
    super(message)
    this.name = 'ApiError'
  }
}

export async function api<T>(path: string, init: RequestInit = {}, token?: string): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body !== undefined) headers.set('Content-Type', 'application/json')
  if (token) headers.set('Authorization', `Bearer ${token}`)
  let response: Response
  try {
    response = await fetch(`${ApiBasePath}${path}`, { ...init, credentials: 'include', headers })
  } catch {
    throw new ApiError(0, ApiErrorCode.Network, '无法连接服务器，请检查网络后重试')
  }
  const payload = (await response.json().catch(() => ({}))) as ApiResponse<T>
  const code = typeof payload.code === 'number' ? payload.code : ApiErrorCode.System
  if (!response.ok || code !== 0) {
    throw new ApiError(response.status, code, payload.message ?? '请求失败，请稍后再试')
  }
  return payload.data as T
}

export function post<T>(path: string, body?: object, token?: string): Promise<T> {
  return api<T>(path, { method: 'POST', ...(body ? { body: JSON.stringify(body) } : {}) }, token)
}
