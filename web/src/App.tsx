import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { setToken } from './api'
import { useLang, useSetLang, useT } from './i18n'
import ReportBugButton from './components/ReportBugButton'

export default function App() {
  const navigate = useNavigate()
  const t = useT()
  const lang = useLang()
  const setLang = useSetLang()
  return (
    <div className="app">
      <header className="topbar">
        <span className="brand">{t('brand')}</span>
        <nav>
          <ReportBugButton />
          <NavLink to="/">{t('nav.games')}</NavLink>
          <NavLink to="/new">{t('nav.new')}</NavLink>
          <NavLink to="/decks">{t('nav.decks')}</NavLink>
          <select
            aria-label="Language"
            value={lang}
            onChange={(e) => setLang(e.target.value as 'en' | 'zh')}
          >
            <option value="en">EN</option>
            <option value="zh">中文</option>
          </select>
          <button
            className="linklike"
            onClick={() => {
              setToken(null)
              navigate('/login')
            }}
          >
            {t('nav.logout')}
          </button>
        </nav>
      </header>
      <main>
        <Outlet />
      </main>
    </div>
  )
}
