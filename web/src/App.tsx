import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { setToken } from './api'

export default function App() {
  const navigate = useNavigate()
  return (
    <div className="app">
      <header className="topbar">
        <span className="brand">Marvel Champions</span>
        <nav>
          <NavLink to="/">Games</NavLink>
          <NavLink to="/new">New Game</NavLink>
          <NavLink to="/decks">Decks</NavLink>
          <button
            className="linklike"
            onClick={() => {
              setToken(null)
              navigate('/login')
            }}
          >
            Log out
          </button>
        </nav>
      </header>
      <main>
        <Outlet />
      </main>
    </div>
  )
}
