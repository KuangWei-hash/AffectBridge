package model

import "time"

// EventStatus tracks whether a narrative occurrence can still be correlated
// with a later confirmation or disconfirmation.
type EventStatus string

const (
	EventStatusPending      EventStatus = "pending"
	EventStatusActive       EventStatus = "active"
	EventStatusRealized     EventStatus = "realized"
	EventStatusDisconfirmed EventStatus = "disconfirmed"
	EventStatusClosed       EventStatus = "closed"
)

func (s EventStatus) Valid() bool {
	switch s {
	case EventStatusPending, EventStatusActive, EventStatusRealized,
		EventStatusDisconfirmed, EventStatusClosed:
		return true
	default:
		return false
	}
}

// EventRelation describes how the current Story relates to a reused Event.
type EventRelation string

const (
	EventRelationNew             EventRelation = "new"
	EventRelationContinuation    EventRelation = "continuation"
	EventRelationConfirmation    EventRelation = "confirmation"
	EventRelationDisconfirmation EventRelation = "disconfirmation"
)

func (r EventRelation) Valid() bool {
	switch r {
	case EventRelationNew, EventRelationContinuation,
		EventRelationConfirmation, EventRelationDisconfirmation:
		return true
	default:
		return false
	}
}

// Topic is a continuing narrative context. It is never sent to ALMA as an
// elicitor; NarrativeEvent.ID is the elicitor identity.
type Topic struct {
	ID               string    `json:"id"`
	CharacterID      string    `json:"character_id"`
	CanonicalSummary string    `json:"canonical_summary"`
	Participants     []string  `json:"participants"`
	CreatedAt        time.Time `json:"created_at"`
	LastSeenAt       time.Time `json:"last_seen_at"`
}

// NarrativeEvent is one occurrence nested under a Topic. ID is stable,
// monotonic, never recycled, and suitable for ALMA's elicitor field.
type NarrativeEvent struct {
	ID               string      `json:"id"`
	TopicID          string      `json:"topic_id"`
	CharacterID      string      `json:"character_id"`
	CanonicalSummary string      `json:"canonical_summary"`
	EventType        string      `json:"event_type"`
	Subject          string      `json:"subject"`
	Target           string      `json:"target"`
	Status           EventStatus `json:"status"`
	CreatedAt        time.Time   `json:"created_at"`
	LastSeenAt       time.Time   `json:"last_seen_at"`
	ResolvedAt       *time.Time  `json:"resolved_at,omitempty"`
}

// TopicMatchDecision is validated before it is allowed to mutate the pool.
// New IDs are always assigned by the repository, never by the LLM.
type TopicMatchDecision struct {
	Decision     string        `json:"decision"`
	Relation     EventRelation `json:"relation"`
	TopicID      string        `json:"topic_id"`
	EventID      string        `json:"event_id"`
	TopicSummary string        `json:"topic_summary"`
	EventSummary string        `json:"event_summary"`
	EventType    string        `json:"event_type"`
	Subject      string        `json:"subject"`
	Target       string        `json:"target"`
	Status       EventStatus   `json:"status"`
	Reason       string        `json:"reason"`
}

// TopicMatchProposal is a non-mutating matcher result tied to one repository
// snapshot. Commit succeeds only if PoolVersion is still current.
type TopicMatchProposal struct {
	CharacterID string             `json:"character_id"`
	PoolVersion uint64             `json:"pool_version"`
	Decision    TopicMatchDecision `json:"decision"`
}

const (
	TopicDecisionReuseEvent    = "reuse_event"
	TopicDecisionNewEvent      = "new_event"
	TopicDecisionNewTopicEvent = "new_topic_event"
)

// TopicResolution is the committed identity used by later appraisal stages.
type TopicResolution struct {
	Topic    Topic          `json:"topic"`
	Event    NarrativeEvent `json:"event"`
	Created  bool           `json:"created"`
	Relation EventRelation  `json:"relation"`
	Reason   string         `json:"reason"`
}
