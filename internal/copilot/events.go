package copilot

import (
	"fmt"

	sdk "github.com/github/copilot-sdk/go"
)

// EventType constants for VibeAuracle event routing.
const (
	EventMessageDelta   = "message_delta"
	EventReasoningDelta = "reasoning_delta"
	EventMessage        = "message"
	EventReasoning      = "reasoning"
	EventToolCall       = "tool_call"
	EventToolResult     = "tool_result"
	EventIdle           = "idle"
	EventError          = "error"
)

// VibeEvent is a simplified event structure for VibeAuracle's UI.
type VibeEvent struct {
	Type    string
	Content string
	ToolID  string // For tool events
	Error   error
}

// EventRouter translates Copilot SDK events to VibeAuracle events.
type EventRouter struct {
	handlers []func(VibeEvent)
}

// NewEventRouter creates a new event router.
func NewEventRouter() *EventRouter {
	return &EventRouter{
		handlers: make([]func(VibeEvent), 0),
	}
}

// OnEvent registers a handler for VibeAuracle events.
func (r *EventRouter) OnEvent(handler func(VibeEvent)) {
	r.handlers = append(r.handlers, handler)
}

// emit sends an event to all registered handlers.
func (r *EventRouter) emit(event VibeEvent) {
	for _, h := range r.handlers {
		h(event)
	}
}

// HandleSDKEvent processes a Copilot SDK event and routes it to VibeAuracle handlers.
func (r *EventRouter) HandleSDKEvent(event sdk.SessionEvent) {
	switch event.Type {
	case "assistant.message_delta":
		if event.Data.DeltaContent != nil {
			r.emit(VibeEvent{
				Type:    EventMessageDelta,
				Content: *event.Data.DeltaContent,
			})
		}

	case "assistant.reasoning_delta":
		if event.Data.DeltaContent != nil {
			r.emit(VibeEvent{
				Type:    EventReasoningDelta,
				Content: *event.Data.DeltaContent,
			})
		}

	case "assistant.message":
		if event.Data.Content != nil {
			r.emit(VibeEvent{
				Type:    EventMessage,
				Content: *event.Data.Content,
			})
		}

	case "assistant.reasoning":
		if event.Data.Content != nil {
			r.emit(VibeEvent{
				Type:    EventReasoning,
				Content: *event.Data.Content,
			})
		}

	case "tool.call":
		// Tool call initiated
		toolID := ""
		if event.Data.ToolCallID != nil {
			toolID = *event.Data.ToolCallID
		}
		r.emit(VibeEvent{
			Type:   EventToolCall,
			ToolID: toolID,
		})

	case "tool.result":
		// Tool execution completed
		toolID := ""
		if event.Data.ToolCallID != nil {
			toolID = *event.Data.ToolCallID
		}
		r.emit(VibeEvent{
			Type:   EventToolResult,
			ToolID: toolID,
		})

	case "session.idle":
		r.emit(VibeEvent{
			Type: EventIdle,
		})

	case "error":
		errMsg := "unknown error"
		if event.Data.Content != nil {
			errMsg = *event.Data.Content
		}
		r.emit(VibeEvent{
			Type:  EventError,
			Error: fmt.Errorf(errMsg),
		})
	}
}
