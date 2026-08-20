import React from 'react'
import './style.css'
import ReactDOM from 'react-dom/client'
import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom'
import App from './App'
import Login from './views/Login'
import Decks from './views/Decks'
import DeckDetail from './views/DeckDetail'
import NewGame from './views/NewGame'
import Games from './views/Games'
import Game from './views/Game'
import { getToken } from './api'
import { LangProvider, getInitialLang } from './i18n'
import { preloadManifest } from './cards'

preloadManifest(getInitialLang())

function RequireAuth({ children }: { children: React.ReactElement }) {
  if (!getToken()) return <Navigate to="/login" replace />
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
      { path: 'games/:id', element: <RequireAuth><Game /></RequireAuth> },
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
