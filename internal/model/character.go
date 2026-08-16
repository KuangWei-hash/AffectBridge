package model

import "time"

// Character aggregates the full affective state of a playable
// character. CreatedAt and UpdatedAt are wall-clock timestamps
// provided by the server.
type Character struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Personality Personality `json:"personality"`
	Mood        Mood        `json:"mood"`
	Emotions    EmotionSet  `json:"emotions"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}
