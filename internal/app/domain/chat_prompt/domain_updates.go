package llmchat

import (
	"strings"
	"sync"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

// DomainUpdateBroker manages per-session SSE listeners for domain-specific mutations.
type DomainUpdateBroker struct {
	mu        sync.RWMutex
	listeners map[string]map[chan models.StreamEvent]string // sessionID -> channel -> domain
}

// NewDomainUpdateBroker creates a new broker.
func NewDomainUpdateBroker() *DomainUpdateBroker {
	return &DomainUpdateBroker{
		listeners: make(map[string]map[chan models.StreamEvent]string),
	}
}

// Subscribe registers a listener for a given session/domain pair and returns the channel plus an unsubscribe func.
func (b *DomainUpdateBroker) Subscribe(sessionID, domain string) (chan models.StreamEvent, func()) {
	ch := make(chan models.StreamEvent, 10)

	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.listeners[sessionID]; !exists {
		b.listeners[sessionID] = make(map[chan models.StreamEvent]string)
	}
	b.listeners[sessionID][ch] = strings.ToLower(domain)

	return ch, func() {
		b.unsubscribe(sessionID, ch)
	}
}

// Publish fan-outs an event to all listeners subscribed to the session/domain.
func (b *DomainUpdateBroker) Publish(sessionID string, event models.StreamEvent) {
	if sessionID == "" {
		return
	}

	b.mu.RLock()
	listeners, exists := b.listeners[sessionID]
	b.mu.RUnlock()
	if !exists || len(listeners) == 0 {
		return
	}

	for ch, domain := range listeners {
		if domain != "" && event.Domain != "" && !strings.EqualFold(domain, event.Domain) {
			continue
		}
		select {
		case ch <- event:
		default:
			// Drop event if listener is slow to avoid blocking critical streaming path.
		}
	}
}

func (b *DomainUpdateBroker) unsubscribe(sessionID string, ch chan models.StreamEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if sessionListeners, exists := b.listeners[sessionID]; exists {
		if _, ok := sessionListeners[ch]; ok {
			delete(sessionListeners, ch)
			close(ch)
		}
		if len(sessionListeners) == 0 {
			delete(b.listeners, sessionID)
		}
	}
}

// DomainUpdates is the shared broker used by handlers to subscribe/publish domain mutations.
var DomainUpdates = NewDomainUpdateBroker()
