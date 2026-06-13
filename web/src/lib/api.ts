import axios from 'axios'

let accessToken: string | null = null
let refreshPromise: Promise<string> | null = null

export const api = axios.create({ baseURL: '/api', withCredentials: true })

export function setAccessToken(token: string | null) {
  accessToken = token
}

export function getAccessToken(): string | null {
  return accessToken
}

api.interceptors.request.use(config => {
  if (accessToken) config.headers.Authorization = `Bearer ${accessToken}`
  return config
})

api.interceptors.response.use(
  res => res,
  async err => {
    const original = err.config
    if (err.response?.status === 401 && !original._retry) {
      original._retry = true
      try {
        if (!refreshPromise) {
          refreshPromise = axios
            .post('/api/auth/refresh', {}, { withCredentials: true })
            .then(res => res.data.access_token as string)
            .finally(() => { refreshPromise = null })
        }
        const newToken = await refreshPromise
        setAccessToken(newToken)
        original.headers.Authorization = `Bearer ${newToken}`
        return api(original)
      } catch {
        setAccessToken(null)
        window.location.href = '/login'
      }
    }
    return Promise.reject(err)
  },
)
