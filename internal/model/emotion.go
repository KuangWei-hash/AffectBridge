package model

// EmotionSet is the short-lived OCC emotion layer used by ALMA.
// Keys follow ALMA's emotion taxonomy (e.g. "anger", "joy", "fear",
// "distress", "hope", "reproach", "admiration", ...). Values are
// intensities in [0, 1].
type EmotionSet map[string]float64
