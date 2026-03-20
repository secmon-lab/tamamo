package event

import "time"

// Event represents an attacker interaction captured by the honeypot.
type Event struct {
	Timestamp time.Time         `json:"timestamp"`
	NodeID    string            `json:"node_id"`
	EventType string            `json:"event_type"`
	SourceIP  string            `json:"source_ip"`
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Headers   map[string]string `json:"headers"`
	Body      any               `json:"body,omitempty"`
	Scenario  string            `json:"scenario"`
}
