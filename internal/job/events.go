package job

import (
	"sync"
)

// EventType represents the type of job event
type EventType string

const (
	EventTypeJobStatusChange   EventType = "job_status_change"
	EventTypeJobProgress       EventType = "job_progress"
	EventTypeImageStatusChange EventType = "image_status_change"
	EventTypeImageCompleted    EventType = "image_completed"
	EventTypeImageFailed       EventType = "image_failed"
)

// Event represents a job or image event
type Event struct {
	Type    EventType
	JobID   int64
	ImageID int64
	Data    map[string]interface{}
}

// EventBus manages event subscriptions and publishing
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[int64][]chan Event // jobID -> []subscriber channels
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[int64][]chan Event),
	}
}

// Subscribe subscribes to events for a specific job
func (eb *EventBus) Subscribe(jobID int64) chan Event {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan Event, 10) // Buffer to avoid blocking publishers
	eb.subscribers[jobID] = append(eb.subscribers[jobID], ch)
	return ch
}

// Unsubscribe removes a subscriber channel
func (eb *EventBus) Unsubscribe(jobID int64, ch chan Event) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subs := eb.subscribers[jobID]
	for i, sub := range subs {
		if sub == ch {
			// Remove this subscriber
			eb.subscribers[jobID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}

	// Clean up empty subscriber lists
	if len(eb.subscribers[jobID]) == 0 {
		delete(eb.subscribers, jobID)
	}
}

// Publish publishes an event to all subscribers of the job
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	subs, exists := eb.subscribers[event.JobID]
	if !exists {
		return
	}

	// Send to all subscribers (non-blocking)
	for _, ch := range subs {
		select {
		case ch <- event:
		default:
			// Subscriber's channel is full, skip
		}
	}
}
