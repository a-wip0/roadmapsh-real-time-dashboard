import { useState, type FormEvent } from 'react'
import { api, type SubmitResult } from '../api'

export function ScoreForm({ token, onSubmitted }: { token: string; onSubmitted: () => void }) {
  const [game, setGame] = useState('')
  const [score, setScore] = useState('')
  const [result, setResult] = useState<SubmitResult | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      setResult(await api.submitScore(token, game.trim(), Number(score)))
      setScore('')
      onSubmitted()
    } catch (err) {
      setResult(null)
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setBusy(false)
    }
  }

  return (
    <form className="card" onSubmit={onSubmit}>
      <h2>Submit a score</h2>
      <div className="row">
        <input
          value={game}
          onChange={(e) => setGame(e.target.value)}
          placeholder="Game"
          pattern="[A-Za-z0-9_.\-]{1,64}"
          title="Letters, digits, _ . - (max 64)"
          aria-label="Game"
          required
        />
        <input
          type="number"
          value={score}
          onChange={(e) => setScore(e.target.value)}
          placeholder="Score"
          min={0}
          step={1}
          aria-label="Score"
          required
        />
        <button type="submit" disabled={busy}>Submit</button>
      </div>
      {error && <p className="error" role="alert">{error}</p>}
      {result && (
        <p className="ok" role="status">
          {result.new_personal_best ? 'New personal best! ' : `Recorded (your best is ${result.personal_best.toLocaleString()}). `}
          Rank #{result.game_rank} in {result.game}, #{result.global_rank} overall.
        </p>
      )}
    </form>
  )
}
