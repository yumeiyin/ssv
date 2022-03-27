package network

import (
	"github.com/bloxapp/ssv/protocol/v1/core"
)

// MessageRouter is accepting network messages and route them to the corresponding (internal) components
type MessageRouter interface {
	// Route routes the given message, this function MUST NOT block
	Route(message core.SSVMessage)
}

// MsgValidationResult helps other components to report message validation with a generic results scheme
type MsgValidationResult int32

const (
	// ValidationAccept is the result of a valid message
	ValidationAccept MsgValidationResult = iota
	// ValidationIgnore is the result in case we want to ignore the validation
	ValidationIgnore
	// ValidationRejectLow is the result for invalid message, with low severity (e.g. late message)
	ValidationRejectLow
	// ValidationRejectMedium is the result for invalid message, with medium severity (e.g. wrong height)
	ValidationRejectMedium
	// ValidationRejectHigh is the result for invalid message, with high severity (e.g. invalid signature)
	ValidationRejectHigh
)

// ValidationReporter is the interface for reporting on message validation results
type ValidationReporter interface {
	// ReportValidation reports the result for the given message
	ReportValidation(message core.SSVMessage, res MsgValidationResult)
}

// SubscriberV1 is the interface for subscribing to topics
type SubscriberV1 interface {
	// Subscribe subscribes to validator subnet
	Subscribe(pk core.ValidatorPK) error
	// Unsubscribe unsubscribes from the validator subnet
	Unsubscribe(pk core.ValidatorPK) error
	// UseMessageRouter registers a message router to handle incoming messages
	UseMessageRouter(router MessageRouter)
}

// BroadcasterV1 is the interface for broadcasting messages
type BroadcasterV1 interface {
	// Broadcast publishes the message to all peers in subnet
	Broadcast(message core.SSVMessage) error
}

// StreamHandler handles a stream request with a simple interface of handling the incoming request
// and providing the answer. the actual work with streams is hidden
type StreamHandler func(*core.SSVMessage) (*core.SSVMessage, error)

// SyncerV1 is the interface for syncing messages
type SyncerV1 interface {
	// LastState fetches last decided from a random set of peers
	LastState(mid core.Identifier) ([]core.SSVMessage, error)
	// GetHistory sync the given range from a set of peers that supports history for the given identifier
	GetHistory(mid core.Identifier, from, to uint64) ([]core.SSVMessage, error)
	// LastChangeRound fetches last change round message from a random set of peers
	LastChangeRound(mid core.Identifier) ([]core.SSVMessage, error)
	// SetStreamHandler registers the given handler for the stream
	SetStreamHandler(protocol string, handler StreamHandler)
}

// V1 is a facade interface that provides the entire functionality of the different network interfaces
type V1 interface {
	ValidationReporter
	SubscriberV1
	BroadcasterV1
	SyncerV1
	Start() error
	Setup() error
}
