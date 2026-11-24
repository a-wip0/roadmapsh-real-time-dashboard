import { useState, type FormEvent } from 'react'
import { useLiveLeaderboard } from '../hooks/useLiveLeaderboard'

export function Leaderboard({ username }: { username: string | null }) {
  const [game, setGame] = useState('')
  const [draft, setDraft] = useState('')
  const { entries, error, live } = useLiveLeaderboard(game)

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    setGame(draft.trim())
  }

  return (
    <section className="card">
      <div className="row">
        <h2>{game ? `Top players — ${game}` : 'Global leaderboard'}</h2>
        <span className={live ? 'pill live' : 'pill'} title="Live updates over Server-Sent Events">
          {live ? '● live' : '○ offline'}
        </span>
      </div>
      <form className="row" onSubmit={onSubmit}>
        <input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder="Game name (empty = all games)"
          pattern="[A-Za-z0-9_.\-]{0,64}"
          title="Letters, digits, _ . -"
          aria-label="Game"
        />
        <button type="submit">Show</button>
      </form>
      {error && <p className="error" role="alert">{error}</p>}
      {entries.length === 0 && !error ? (
        <p className="muted">No scores yet.</p>
      ) : (
        <table>
          <thead>
            <tr><th>#</th><th>Player</th><th className="num">Score</th></tr>
          </thead>
          <tbody>
            {entries.map((e) => (
              <tr key={e.username} className={e.username === username ? 'me' : undefined}>
                <td>{e.rank}</td>
                <td>{e.username}</td>
                <td className="num">{e.score.toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  )
}
