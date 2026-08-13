/**
 * API client for the Go (Gin) backend.
 *
 * Response envelope: { code, message, data }  (internal/pkg/response + internal/api).
 * Requests hit `/api/*`; in dev, Vite proxies that to http://localhost:8080.
 * The JWT is stored in localStorage and sent as `Authorization: Bearer <token>`.
 */

export interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
}

export class ApiError extends Error {
  status: number
  code?: number
  constructor(message: string, status: number, code?: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

const BASE = '/api'
const TOKEN_KEY = 'fina.token'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string | null): void {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

async function request<T>(
  method: string,
  path: string,
  payload?: unknown,
): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (payload !== undefined) headers['Content-Type'] = 'application/json'
  const token = getToken()
  if (token) headers.Authorization = `Bearer ${token}`

  let res: Response
  try {
    res = await fetch(`${BASE}${path}`, {
      method,
      headers,
      body: payload !== undefined ? JSON.stringify(payload) : undefined,
    })
  } catch {
    throw new ApiError('Không kết nối được tới máy chủ. Backend có đang chạy không?', 0)
  }

  const isJson = (res.headers.get('content-type') || '').includes('application/json')
  let body: ApiEnvelope<T> | null = null
  if (isJson) {
    try {
      body = (await res.json()) as ApiEnvelope<T>
    } catch {
      body = null
    }
  }

  if (!res.ok) {
    throw new ApiError(body?.message || `Yêu cầu thất bại (${res.status})`, res.status, body?.code)
  }
  if (!body) {
    // 2xx but not a JSON envelope (e.g. proxy returned index.html) — surface a
    // clear message instead of a raw "reading 'data' of null" TypeError.
    throw new ApiError('Máy chủ trả về phản hồi không hợp lệ (không phải JSON API).', res.status)
  }
  return body.data
}

export const apiGet = <T>(path: string) => request<T>('GET', path)
export const apiSend = <T>(
  method: 'POST' | 'PUT' | 'DELETE' | 'PATCH',
  path: string,
  payload?: unknown,
) => request<T>(method, path, payload)
