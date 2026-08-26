// 顶栏右上角用户菜单：显示当前用户名，展开后可退出登录。
// 用户名来自 /whoami（token 里只有 userID）；未登录时不渲染。
import { useEffect, useRef, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { get, getToken, setToken } from '../api'
import { useT } from '../i18n'

export default function UserMenu() {
  const t = useT()
  const navigate = useNavigate()
  const location = useLocation()
  const [name, setName] = useState<string | null>(null)
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement | null>(null)

  // 登录/登出都伴随路由跳转：每次导航后按当前 token 同步一次身份。
  useEffect(() => {
    let cancelled = false
    if (!getToken()) {
      setName(null)
      setOpen(false)
      return
    }
    get<{ id: number; username: string }>('/whoami')
      .then((me) => {
        if (!cancelled) setName(me.username)
      })
      .catch(() => {
        /* 401 已由 api 层重定向到登录页 */
      })
    return () => {
      cancelled = true
    }
  }, [location.pathname])

  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  function logout() {
    setOpen(false)
    setToken(null)
    navigate('/login')
  }

  if (!getToken()) return null
  return (
    <div className="user-menu" ref={rootRef}>
      <button
        type="button"
        className="user-menu-btn"
        onClick={() => setOpen(!open)}
        aria-haspopup="menu"
        aria-expanded={open}
      >
        {name ?? '…'}
        <span className="user-menu-caret">▾</span>
      </button>
      {open && (
        <div className="user-menu-panel" role="menu">
          <button type="button" role="menuitem" onClick={logout}>
            {t('nav.logout')}
          </button>
        </div>
      )}
    </div>
  )
}
