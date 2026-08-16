// Package events provides a versioned, append-only Event Envelope store.
//
// Commands (intent to change state) and Events (immutable facts) are strictly separate.
// This package only records Events. No Kafka or distributed bus in Phase 1.
package events
