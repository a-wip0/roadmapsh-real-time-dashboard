import { AuthForm } from './components/AuthForm'
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
      <main>{token ? <p>Signed in as {username}.</p> : <AuthForm />}</main>
    </>
  )
}
