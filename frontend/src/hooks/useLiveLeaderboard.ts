import { useEffect, useState } from 'react'
import { api, type Entry } from '../api'

/**
 * Top entries for the global board (game = '') or one game.
 * The backend pushes the global top 10 over SSE whenever someone sets a new
 * personal best. For the global board that snapshot is used directly; for a
 * single game it is treated as a "something changed" signal and the game
 * board is refetched.
 */
export function useLiveLeaderboard(game: string) {
  const [entries, setEntries] = useState<Entry[]>([])
  const [error, setError] = useState('')
  const [live, setLive] = useState(false)

  useEffect(() => {
    let cancelled = false
    const load = () =>
      api
        .leaderboard(game || undefined)
        .then((e) => {
          if (cancelled) return
          setEntries(e)
          setError('')
        })
        .catch((err: Error) => !cancelled && setError(err.message))
    load()

    const es = new EventSource('/api/leaderboard/stream')
    es.onopen = () => setLive(true)
    es.onerror = () => setLive(false) // EventSource reconnects on its own
    es.addEventListener('leaderboard', (ev) => {
      if (game) load()
      else setEntries(JSON.parse((ev as MessageEvent<string>).data) as Entry[])
    })
    return () => {
      cancelled = true
      es.close()
      setLive(false)
    }
  }, [game])

  return { entries, error, live }
}
