import React from 'react'
import './style.css'
import ReactDOM from 'react-dom/client'
import { createBrowserRouter, RouterProvider, Navigate, useLocation } from 'react-router-dom'
import App from './App'
import Login from './views/Login'
import Decks from './views/Decks'
import DeckDetail from './views/DeckDetail'
import NewGame from './views/NewGame'
import Games from './views/Games'
import GamePage from './views/GamePage'
import { getToken } from './api'
import { LangProvider, getInitialLang } from './i18n'
import { preloadManifest } from './cards'

preloadManifest(getInitialLang())

function RequireAuth({ children }: { children: React.ReactElement }) {
  const location = useLocation()
  if (!getToken()) return <Navigate to="/login" replace state={{ from: location.pathname }} />
  return children
}

const router = createBrowserRouter([
  {
    path: '/',
    element: <App />,
    children: [
      { index: true, element: <RequireAuth><Games /></RequireAuth> },
      { path: 'login', element: <Login /> },
      { path: 'decks', element: <RequireAuth><Decks /></RequireAuth> },
      { path: 'decks/:id', element: <RequireAuth><DeckDetail /></RequireAuth> },
      { path: 'new', element: <RequireAuth><NewGame /></RequireAuth> },
      { path: 'games/:id', element: <RequireAuth><GamePage /></RequireAuth> },
    ],
  },
])

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <LangProvider>
      <RouterProvider router={router} />
    </LangProvider>
  </React.StrictMode>,
)
