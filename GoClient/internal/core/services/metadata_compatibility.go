package services

import (
	"fmt"
	"strings"

	"weicloth/internal/core/domain"
)

var seasonAdjacency = map[string][]string{
	"summer":     {"spring", "fall"},
	"spring":     {"summer", "fall"},
	"fall":       {"spring", "winter"},
	"winter":     {"fall"},
	"all_season": {"summer", "winter", "spring", "fall"},
}

var occasionCompat = map[string]map[string]float64{
	"casual":  {"casual": 1.0, "outdoor": 0.8, "beach": 0.8, "sport": 0.6, "formal": 0.4, "party": 0.5},
	"formal":  {"formal": 1.0, "party": 0.7, "casual": 0.4, "outdoor": 0.3, "sport": 0.2, "beach": 0.2},
	"sport":   {"sport": 1.0, "outdoor": 0.7, "casual": 0.6, "beach": 0.5, "formal": 0.2, "party": 0.3},
	"party":   {"party": 1.0, "formal": 0.7, "casual": 0.5, "outdoor": 0.4, "sport": 0.3, "beach": 0.5},
	"outdoor": {"outdoor": 1.0, "casual": 0.8, "sport": 0.7, "beach": 0.6, "formal": 0.3, "party": 0.4},
	"beach":   {"beach": 1.0, "casual": 0.8, "outdoor": 0.6, "sport": 0.5, "party": 0.5, "formal": 0.2},
}

var materialCompat = map[string]map[string]float64{
	"cotton":    {"cotton": 1.0, "denim": 0.9, "linen": 0.9, "wool": 0.7, "polyester": 0.8, "leather": 0.6, "silk": 0.7, "suede": 0.6},
	"denim":     {"denim": 1.0, "cotton": 0.9, "leather": 0.8, "wool": 0.7, "linen": 0.7, "polyester": 0.7, "silk": 0.5, "suede": 0.7},
	"leather":   {"leather": 1.0, "denim": 0.8, "wool": 0.7, "cotton": 0.6, "suede": 0.8, "silk": 0.5, "polyester": 0.5, "linen": 0.4},
	"silk":      {"silk": 1.0, "wool": 0.7, "cotton": 0.7, "polyester": 0.6, "leather": 0.5, "denim": 0.5, "linen": 0.7, "suede": 0.6},
	"wool":      {"wool": 1.0, "cotton": 0.7, "denim": 0.7, "leather": 0.7, "silk": 0.7, "suede": 0.8, "linen": 0.5, "polyester": 0.6},
	"polyester": {"polyester": 1.0, "cotton": 0.8, "denim": 0.7, "wool": 0.6, "linen": 0.6, "leather": 0.5, "silk": 0.6, "suede": 0.5},
	"linen":     {"linen": 1.0, "cotton": 0.9, "denim": 0.7, "silk": 0.7, "wool": 0.5, "leather": 0.4, "polyester": 0.6, "suede": 0.5},
	"suede":     {"suede": 1.0, "leather": 0.8, "wool": 0.8, "denim": 0.7, "cotton": 0.6, "silk": 0.6, "linen": 0.5, "polyester": 0.5},
}

func normMeta(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func SeasonCompatibilityScore(a, b string) float64 {
	a, b = normMeta(a), normMeta(b)
	if a == "" || b == "" {
		return 0.7
	}
	if a == b || a == "all_season" || b == "all_season" {
		return 1.0
	}
	if isAdjacentSeason(a, b) {
		return 0.7
	}
	return 0.3
}

func isAdjacentSeason(a, b string) bool {
	for _, adj := range seasonAdjacency[a] {
		if adj == b {
			return true
		}
	}
	return false
}

func OutfitSeasonScore(items []domain.Garment) (float64, string) {
	seasons := collectMeta(items, func(g domain.Garment) string { return g.Season })
	if len(seasons) == 0 {
		return 0.7, "season metadata unavailable"
	}
	score := minPairwiseScore(seasons, SeasonCompatibilityScore)
	return score, fmt.Sprintf("season coherence: %s", strings.Join(uniqueNonEmpty(seasons), ", "))
}

func OccasionCompatibilityScore(a, b string) float64 {
	a, b = normMeta(a), normMeta(b)
	if a == "" || b == "" {
		return 0.7
	}
	if row, ok := occasionCompat[a]; ok {
		if s, ok := row[b]; ok {
			return s
		}
	}
	if a == b {
		return 1.0
	}
	return 0.4
}

func OutfitOccasionScore(items []domain.Garment) (float64, string) {
	occasions := collectMeta(items, func(g domain.Garment) string { return g.Occasion })
	if len(occasions) == 0 {
		return 0.7, "occasion metadata unavailable"
	}
	score := minPairwiseScore(occasions, OccasionCompatibilityScore)
	return score, fmt.Sprintf("occasion match: %s", strings.Join(uniqueNonEmpty(occasions), ", "))
}

func MaterialCompatibilityScore(a, b string) float64 {
	a, b = normMeta(a), normMeta(b)
	if a == "" || b == "" {
		return 0.7
	}
	if a == b {
		return 1.0
	}
	if row, ok := materialCompat[a]; ok {
		if s, ok := row[b]; ok {
			return s
		}
	}
	return 0.5
}

func OutfitMaterialScore(items []domain.Garment) (float64, string) {
	materials := collectMeta(items, func(g domain.Garment) string { return g.Material })
	if len(materials) == 0 {
		return 0.7, ""
	}
	score := avgPairwiseScore(materials, MaterialCompatibilityScore)
	return score, fmt.Sprintf("material blend: %s", strings.Join(uniqueNonEmpty(materials), ", "))
}

var patternedTypes = map[string]bool{
	"striped": true, "checkered": true, "floral": true, "polka_dots": true,
	"camouflage": true, "graphic": true, "animal_print": true,
}

func isPatterned(pattern string) bool {
	return patternedTypes[normMeta(pattern)]
}

// OutfitPatternScore penalizes multiple patterned items in one outfit.
func OutfitPatternScore(items []domain.Garment) (float64, string) {
	patterns := collectMeta(items, func(g domain.Garment) string { return g.Pattern })
	if len(patterns) == 0 {
		return 0.8, "pattern metadata unavailable"
	}

	patternedCount := 0
	var patternedNames []string
	for _, p := range patterns {
		if isPatterned(p) {
			patternedCount++
			patternedNames = append(patternedNames, p)
		}
	}

	switch {
	case patternedCount == 0:
		return 1.0, "solid palette throughout"
	case patternedCount == 1:
		return 1.0, fmt.Sprintf("single patterned item (%s)", patternedNames[0])
	default:
		return 0.3, fmt.Sprintf("multiple patterns (%s)", strings.Join(patternedNames, ", "))
	}
}

func collectMeta(items []domain.Garment, fn func(domain.Garment) string) []string {
	var out []string
	for _, g := range items {
		v := normMeta(fn(g))
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func minPairwiseScore(values []string, scorer func(string, string) float64) float64 {
	if len(values) < 2 {
		return 1.0
	}
	min := 1.0
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			s := scorer(values[i], values[j])
			if s < min {
				min = s
			}
		}
	}
	return min
}

func avgPairwiseScore(values []string, scorer func(string, string) float64) float64 {
	if len(values) < 2 {
		return 1.0
	}
	var sum float64
	var n int
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			sum += scorer(values[i], values[j])
			n++
		}
	}
	if n == 0 {
		return 1.0
	}
	return sum / float64(n)
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, v := range values {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

// garmentMatchesFilter returns true if garment season/occasion matches optional filters.
func garmentMatchesFilter(g domain.Garment, seasonFilter, occasionFilter string) bool {
	seasonFilter = normMeta(seasonFilter)
	occasionFilter = normMeta(occasionFilter)

	if seasonFilter != "" {
		gs := normMeta(g.Season)
		if gs != "" && gs != seasonFilter && gs != "all_season" && seasonFilter != "all_season" {
			if SeasonCompatibilityScore(gs, seasonFilter) < 0.7 {
				return false
			}
		}
	}
	if occasionFilter != "" {
		go_ := normMeta(g.Occasion)
		if go_ != "" && go_ != occasionFilter {
			if OccasionCompatibilityScore(go_, occasionFilter) < 0.6 {
				return false
			}
		}
	}
	return true
}
