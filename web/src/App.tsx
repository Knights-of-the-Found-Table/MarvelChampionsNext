import { Outlet, NavLink } from 'react-router-dom'
import { useLang, useSetLang, useT } from './i18n'
import HotSeat from './components/HotSeat'
import ReportBugButton from './components/ReportBugButton'
import UserMenu from './components/UserMenu'

export default function App() {
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
          <UserMenu />
        </nav>
      </header>
      <main>
        <Outlet />
      </main>
      <HotSeat />
    </div>
  )
}
