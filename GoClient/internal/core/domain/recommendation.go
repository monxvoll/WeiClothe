package domain

// UserStylePreferences stores per-user palette and style filters for recommendations.
type UserStylePreferences struct {
	UserID            string   `json:"user_id"`
	PreferredColors   []string `json:"preferred_colors"`
	PreferredOccasions []string `json:"preferred_occasions"`
	PreferredSeasons  []string `json:"preferred_seasons"`
	AvoidColors       []string `json:"avoid_colors"`
}

// RecommendationRequest configures outfit generation and optional filters.
type RecommendationRequest struct {
	UserID   string
	Season   string // optional: summer, winter, spring, fall, all_season
	Occasion string // optional: casual, formal, sport, party, outdoor, beach
	Limit    int    // max outfits to return (default 3)
}

// OutfitReasoning explains why an outfit scored well.
type OutfitReasoning struct {
	ColorHarmony  string  `json:"color_harmony"`
	SeasonMatch   string  `json:"season_match"`
	OccasionMatch string  `json:"occasion_match"`
	PatternNote   string  `json:"pattern_note"`
	MaterialNote  string  `json:"material_note,omitempty"`
	PreferenceNote string `json:"preference_note,omitempty"`
}

// ScoreWeights defines the composite scoring weights (must sum to 1.0).
type ScoreWeights struct {
	Color     float64
	Season    float64
	Occasion  float64
	Material  float64
	Pattern   float64
	Preference float64
}

// DefaultScoreWeights returns the standard weight distribution.
func DefaultScoreWeights() ScoreWeights {
	return ScoreWeights{
		Color:      0.35,
		Season:     0.20,
		Occasion:   0.20,
		Material:   0.10,
		Pattern:    0.10,
		Preference: 0.05,
	}
}

// ScoredOutfit is an internal ranked candidate before mapping to OutfitRecommendation.
type ScoredOutfit struct {
	Top      Garment
	Bottom   Garment
	Footwear Garment
	Score    float64
	Reasoning OutfitReasoning
}

// MLColorLabels are the canonical color values produced by the vusion-ml CLIP pipeline.
var MLColorLabels = []string{
	"red", "blue", "black", "white", "green", "yellow", "pink", "orange",
	"purple", "brown", "gray", "beige", "navy", "multicolor",
}
