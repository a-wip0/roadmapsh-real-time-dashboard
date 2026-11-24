import { useEffect, useState } from 'react'
import { api, type Entry, type Period } from '../api'

const PERIODS: Period[] = ['day', 'week', 'month']

// `refreshKey` changes after every submission so the report reloads.
export function Report({ refreshKey }: { refreshKey: number }) {
  const [period, setPeriod] = useState<Period>('week')
  const [game, setGame] = useState('')
  const [players, setPlayers] = useState<Entry[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    api
      .report(period, game.trim() || undefined)
      .then((r) => {
        if (cancelled) return
        setPlayers(r.players)
        setError('')
      })
      .catch((err: Error) => !cancelled && setError(err.message))
    return () => {
      cancelled = true
    }
  }, [period, game, refreshKey])

  return (
    <section className="card">
      <h2>Top players report</h2>
      <div className="row">
        <select value={period} onChange={(e) => setPeriod(e.target.value as Period)} aria-label="Period">
          {PERIODS.map((p) => (
            <option key={p} value={p}>
              Last {p}
            </option>
          ))}
        </select>
        <input
          value={game}
          onChange={(e) => setGame(e.target.value)}
          placeholder="Game (empty = all)"
          pattern="[A-Za-z0-9_.\-]{0,64}"
          aria-label="Game"
        />
      </div>
      {error && <p className="error" role="alert">{error}</p>}
      {players.length === 0 && !error ? (
        <p className="muted">No scores in this period.</p>
      ) : (
        <table>
          <thead>
            <tr><th>#</th><th>Player</th><th className="num">Total</th></tr>
          </thead>
          <tbody>
            {players.map((p) => (
              <tr key={p.username}>
                <td>{p.rank}</td>
                <td>{p.username}</td>
                <td className="num">{p.score.toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  )
}
