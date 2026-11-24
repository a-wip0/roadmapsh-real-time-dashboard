import { AuthForm } from './components/AuthForm'
import { Leaderboard } from './components/Leaderboard'
import { useAuth } from './auth-context'

export default function App() {
  const { token, username, logout } = useAuth()
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
        {!token && <AuthForm />}
      </main>
    </>
  )
}
