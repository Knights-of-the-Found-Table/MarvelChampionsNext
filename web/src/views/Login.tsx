import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { post, setToken } from '../api'
import { useT } from '../i18n'

export default function Login() {
  const navigate = useNavigate()
  const location = useLocation()
  const t = useT()
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      const path = mode === 'login' ? '/authenticate' : '/register'
      const resp = await post<{ token: string }>(path, { username, password })
      setToken(resp.token)
      // 登录前访问的路径（如朋友发来的邀请链接）登录后原地返回。
      navigate((location.state as { from?: string } | null)?.from ?? '/')
    } catch (err) {
      setError(String((err as Error).message || err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="login-wrap">
      <form className="login card" onSubmit={submit}>
        <h2>{mode === 'login' ? t('login.title') : t('login.registerTitle')}</h2>
        {error && <p className="error">{error}</p>}
        <label>
          {t('login.username')}
          <input value={username} onChange={(e) => setUsername(e.target.value)} autoFocus minLength={3} required />
        </label>
        <label>
          {t('login.password')}
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} minLength={6} required />
        </label>
        <button type="submit" disabled={busy}>
          {busy ? '…' : mode === 'login' ? t('login.title') : t('login.register')}
        </button>
        <p className="muted">
          <button
            type="button"
            className="linklike"
            onClick={() => setMode(mode === 'login' ? 'register' : 'login')}
          >
            {mode === 'login' ? t('login.toRegister') : t('login.toLogin')}
          </button>
        </p>
      </form>
    </div>
  )
}
