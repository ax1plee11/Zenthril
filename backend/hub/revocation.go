package hub

// DisconnectUser closes this process's realtime connections after account-wide
// revocation. Each connection's read pump performs normal unregister cleanup.
func (h *Hub) DisconnectUser(userID string) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.users[userID]))
	for c := range h.users[userID] {
		clients = append(clients, c)
	}
	h.mu.RUnlock()
	for _, c := range clients {
		if c.conn != nil {
			_ = c.conn.Close()
		}
	}
}
