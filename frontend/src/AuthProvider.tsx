import { useCallback, useMemo, useState, type ReactNode } from 'react'
import { api } from './api'
import { AuthContext } from './auth-context'

const KEY = 'leaderboard.session'

interface Session {
  token: string
  username: string
}

// localStorage can throw (private mode, blocked storage), so never trust it.
function load(): Session | null {
  try {
    const raw = localStorage.getItem(KEY)
    return raw ? (JSON.parse(raw) as Session) : null
  } catch {
    return null
  }
}

function save(s: Session | null) {
  try {
    if (s) localStorage.setItem(KEY, JSON.stringify(s))
    else localStorage.removeItem(KEY)
  } catch {
    /* session just won't persist */
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(load)

  const login = useCallback(async (username: string, password: string) => {
    const { token } = await api.login(username, password)
    const s = { token, username }
    save(s)
    setSession(s)
  }, [])

  const register = useCallback(
    async (username: string, password: string) => {
      await api.register(username, password)
      await login(username, password)
    },
    [login],
  )

  const logout = useCallback(() => {
    save(null)
    setSession(null)
  }, [])

  const value = useMemo(
    () => ({ token: session?.token ?? null, username: session?.username ?? null, login, register, logout }),
    [session, login, register, logout],
  )
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
