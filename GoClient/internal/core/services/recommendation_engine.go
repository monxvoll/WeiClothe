package services

import (
	"fmt"
	"sort"
	"strings"

	"weicloth/internal/core/domain"
)

const defaultRecommendationLimit = 3

// RecommendationEngine scores outfit candidates using metadata harmony rules.
type RecommendationEngine struct {
	weights domain.ScoreWeights
}

// NewRecommendationEngine builds an engine with default weights.
func NewRecommendationEngine() *RecommendationEngine {
	return &RecommendationEngine{weights: domain.DefaultScoreWeights()}
}

// NewRecommendationEngineWithWeights builds an engine with custom weights.
func NewRecommendationEngineWithWeights(w domain.ScoreWeights) *RecommendationEngine {
	return &RecommendationEngine{weights: w}
}

// Recommend returns top-scoring outfits from the wardrobe.
func (e *RecommendationEngine) Recommend(
	req domain.RecommendationRequest,
	garments []domain.Garment,
	prefs *domain.UserStylePreferences,
) []domain.OutfitRecommendation {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultRecommendationLimit
	}

	tops, bottoms, shoes := bucketGarments(garments, req.Season, req.Occasion)
	if len(tops) == 0 || len(bottoms) == 0 || len(shoes) == 0 {
		return nil
	}

	var scored []domain.ScoredOutfit
	for _, top := range tops {
		for _, bottom := range bottoms {
			for _, shoe := range shoes {
				if prefs != nil && outfitHasAvoidedColor(top, bottom, shoe, prefs.AvoidColors) {
					continue
				}
				s, reasoning := e.scoreOutfit(top, bottom, shoe, prefs)
				scored = append(scored, domain.ScoredOutfit{
					Top: top, Bottom: bottom, Footwear: shoe,
					Score: s, Reasoning: reasoning,
				})
			}
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	if len(scored) > limit {
		scored = scored[:limit]
	}

	out := make([]domain.OutfitRecommendation, 0, len(scored))
	for i, s := range scored {
		out = append(out, domain.OutfitRecommendation{
			ID:          fmt.Sprintf("rec-%s-%d", req.UserID, i+1),
			Name:        outfitName(i+1, s.Reasoning),
			Top:         s.Top,
			Bottom:      s.Bottom,
			Footwear:    s.Footwear,
			Score:       s.Score,
			Reasoning:   s.Reasoning,
			Description: buildDescription(s.Score, s.Reasoning),
		})
	}
	return out
}

func bucketGarments(garments []domain.Garment, seasonFilter, occasionFilter string) (tops, bottoms, shoes []domain.Garment) {
	for _, g := range garments {
		if g.Status != "completed" {
			continue
		}
		if !garmentMatchesFilter(g, seasonFilter, occasionFilter) {
			continue
		}

		category := normMeta(g.Category)
		if category == "" {
			category = normMeta(g.GarmentType)
		}

		switch category {
		case "shirt", "jacket", "camiseta", "top", "dress":
			tops = append(tops, g)
		case "pants", "pantalón", "bottom":
			bottoms = append(bottoms, g)
		case "shoes", "calzado":
			shoes = append(shoes, g)
		}
	}
	return tops, bottoms, shoes
}

func (e *RecommendationEngine) scoreOutfit(
	top, bottom, shoe domain.Garment,
	prefs *domain.UserStylePreferences,
) (float64, domain.OutfitReasoning) {
	items := []domain.Garment{top, bottom, shoe}

	colorScore, colorNote := OutfitColorHarmonyScore(top.Color, bottom.Color, shoe.Color)
	seasonScore, seasonNote := OutfitSeasonScore(items)
	occasionScore, occasionNote := OutfitOccasionScore(items)
	materialScore, materialNote := OutfitMaterialScore(items)
	patternScore, patternNote := OutfitPatternScore(items)
	prefScore, prefNote := userPreferenceScore(items, prefs)

	w := e.weights
	total := w.Color*colorScore +
		w.Season*seasonScore +
		w.Occasion*occasionScore +
		w.Material*materialScore +
		w.Pattern*patternScore +
		w.Preference*prefScore

	reasoning := domain.OutfitReasoning{
		ColorHarmony:   colorNote,
		SeasonMatch:    seasonNote,
		OccasionMatch:  occasionNote,
		PatternNote:    patternNote,
		MaterialNote:   materialNote,
		PreferenceNote: prefNote,
	}
	return total, reasoning
}

func userPreferenceScore(items []domain.Garment, prefs *domain.UserStylePreferences) (float64, string) {
	if prefs == nil {
		return 0.7, ""
	}

	preferred := toSet(prefs.PreferredColors)
	if len(preferred) == 0 && len(prefs.PreferredSeasons) == 0 && len(prefs.PreferredOccasions) == 0 {
		return 0.7, ""
	}

	var hits, checks float64
	var notes []string

	for _, g := range items {
		c := normMeta(g.Color)
		if c != "" && len(preferred) > 0 {
			checks++
			if preferred[c] {
				hits++
			}
		}
	}

	seasonSet := toSet(prefs.PreferredSeasons)
	for _, g := range items {
		s := normMeta(g.Season)
		if s != "" && len(seasonSet) > 0 {
			checks++
			if seasonSet[s] || s == "all_season" {
				hits++
			}
		}
	}

	occSet := toSet(prefs.PreferredOccasions)
	for _, g := range items {
		o := normMeta(g.Occasion)
		if o != "" && len(occSet) > 0 {
			checks++
			if occSet[o] {
				hits++
			}
		}
	}

	if checks == 0 {
		return 0.7, ""
	}
	score := hits / checks
	if score >= 0.8 {
		notes = append(notes, "matches your style preferences")
	} else if score >= 0.5 {
		notes = append(notes, "partially matches your preferences")
	}
	return score, strings.Join(notes, "; ")
}

func outfitHasAvoidedColor(top, bottom, shoe domain.Garment, avoid []string) bool {
	avoidSet := toSet(avoid)
	if len(avoidSet) == 0 {
		return false
	}
	for _, c := range []string{top.Color, bottom.Color, shoe.Color} {
		if avoidSet[normMeta(c)] {
			return true
		}
	}
	return false
}

func toSet(values []string) map[string]bool {
	m := make(map[string]bool, len(values))
	for _, v := range values {
		v = normMeta(v)
		if v != "" {
			m[v] = true
		}
	}
	return m
}

func outfitName(rank int, r domain.OutfitReasoning) string {
	base := fmt.Sprintf("Outfit #%d", rank)
	if strings.Contains(r.SeasonMatch, "summer") {
		return "Summer " + base
	}
	if strings.Contains(r.OccasionMatch, "formal") {
		return "Formal " + base
	}
	if strings.Contains(r.OccasionMatch, "casual") {
		return "Casual " + base
	}
	return base
}

func buildDescription(score float64, r domain.OutfitReasoning) string {
	return fmt.Sprintf("Score %.0f%% — %s", score*100, r.ColorHarmony)
}
