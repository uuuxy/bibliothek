package plugins

import (
	"context"
	"log"
	"sync"
)

// EventType is a typed string for event names that plugins can hook into.
type EventType string

const (
	// EventBookReturned is dispatched when a physical copy is returned.
	// Payload type is BookReturnedPayload.
	EventBookReturned EventType = "OnBookReturned"
)

// BookReturnedPayload represents the data context for the EventBookReturned event.
type BookReturnedPayload struct {
	CopyID       string  `json:"copy_id"`
	BarcodeID    string  `json:"barcode_id"`
	Titel        string  `json:"titel"`
	SchuelerID   *string `json:"schueler_id,omitempty"`
	BearbeiterID string  `json:"bearbeiter_id"`
}

// HookFunc is the function signature for plugin hook callbacks.
type HookFunc func(ctx context.Context, payload any) error

// EventRegistry coordinates event listeners and triggers callbacks thread-safely.
type EventRegistry struct {
	mu        sync.RWMutex
	listeners map[EventType][]HookFunc
}

var (
	globalRegistry = &EventRegistry{
		listeners: make(map[EventType][]HookFunc),
	}
)

// RegisterHook registers a callback function for a specific event type.
func RegisterHook(event EventType, hook HookFunc) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.listeners[event] = append(globalRegistry.listeners[event], hook)
	log.Printf("Plugins: Registered hook callback for event: %s", event)
}

// DispatchEvent dispatches an event to all registered listeners.
// Each listener is executed in its own goroutine to avoid blocking the main server execution paths.
func DispatchEvent(ctx context.Context, event EventType, payload any) {
	globalRegistry.mu.RLock()
	listeners, ok := globalRegistry.listeners[event]
	globalRegistry.mu.RUnlock()

	if !ok || len(listeners) == 0 {
		return
	}

	log.Printf("Plugins: Dispatching event %s to %d listener(s)", event, len(listeners))

	for _, listener := range listeners {
		go func(l HookFunc) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Plugins: Panic caught inside listener for event %s: %v", event, r)
				}
			}()

			// Decouple from request lifecycle using WithoutCancel (available in Go 1.21+)
			hookCtx := context.WithoutCancel(ctx)
			if err := l(hookCtx, payload); err != nil {
				log.Printf("Plugins: Hook error for event %s: %v", event, err)
			}
		}(listener)
	}
}
