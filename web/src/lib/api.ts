import axios from 'axios'

let accessToken: string | null = null

export const api = axios.create({ baseURL: '/api', withCredentials: true })

export function setAccessToken(token: string | null) {
  accessToken = token
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
        const res = await axios.post('/api/auth/refresh', {}, { withCredentials: true })
        setAccessToken(res.data.access_token)
        original.headers.Authorization = `Bearer ${res.data.access_token}`
        return api(original)
      } catch {
        setAccessToken(null)
        window.location.href = '/login'
      }
    }
    return Promise.reject(err)
  },
)
