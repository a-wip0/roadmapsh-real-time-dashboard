import { useEffect, useState } from 'react'
import { ApiError, api, type Entry, type HistoryItem } from '../api'

// `refreshKey` changes after every submission so the panel reloads.
export function MyActivity({ token, refreshKey }: { token: string; refreshKey: number }) {
  const [rank, setRank] = useState<Entry | null>(null)
  const [history, setHistory] = useState<HistoryItem[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    Promise.all([
      api.myRank(token).catch((err) => {
        if (err instanceof ApiError && err.status === 404) return null // nothing submitted yet
        throw err
      }),
      api.history(token),
    ])
      .then(([r, h]) => {
        if (cancelled) return
        setRank(r)
        setHistory(h)
        setError('')
      })
      .catch((err: Error) => !cancelled && setError(err.message))
    return () => {
      cancelled = true
    }
  }, [token, refreshKey])

  return (
    <section className="card">
      <h2>Your activity</h2>
      {error && <p className="error" role="alert">{error}</p>}
      <p className="muted">
        {rank ? `Global rank #${rank.rank} with ${rank.score.toLocaleString()} points` : 'No scores submitted yet.'}
      </p>
      {history.length > 0 && (
        <table>
          <thead>
            <tr><th>Game</th><th className="num">Score</th><th>When</th></tr>
          </thead>
          <tbody>
            {history.slice(0, 10).map((h, i) => (
              <tr key={i}>
                <td>{h.game}</td>
                <td className="num">{h.score.toLocaleString()}</td>
                <td>{new Date(h.created_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  )
}
