package app

import "sync"

// hub fans out leaderboard snapshots to connected SSE clients (in-process).
type hub struct {
	mu   sync.Mutex
	subs map[chan []byte]struct{}
}

func newHub() *hub { return &hub{subs: map[chan []byte]struct{}{}} }

func (h *hub) subscribe() chan []byte {
	ch := make(chan []byte, 8)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *hub) unsubscribe(ch chan []byte) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
}

func (h *hub) publish(msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- msg:
		default: // slow client: drop update
		}
	}
}
