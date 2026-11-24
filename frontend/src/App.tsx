import { useState } from 'react'
import { AuthForm } from './components/AuthForm'
import { Leaderboard } from './components/Leaderboard'
import { MyActivity } from './components/MyActivity'
import { Report } from './components/Report'
import { ScoreForm } from './components/ScoreForm'
import { useAuth } from './auth-context'

export default function App() {
  const { token, username, logout } = useAuth()
  const [refreshKey, setRefreshKey] = useState(0)
  return (
    <>
      <header className="top">
        <h1>Leaderboard</h1>
        {token && (
          <span>
            {username} <button className="link" onClick={logout}>Log out</button>
          </span>
        )}
      </header>
      <main>
        <Leaderboard username={username} />
        {token ? (
          <>
            <ScoreForm token={token} onSubmitted={() => setRefreshKey((k) => k + 1)} />
            <MyActivity token={token} refreshKey={refreshKey} />
          </>
        ) : (
          <AuthForm />
        )}
        <Report refreshKey={refreshKey} />
      </main>
    </>
  )
}
