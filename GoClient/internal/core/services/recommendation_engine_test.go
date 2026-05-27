package services

import (
	"testing"

	"weicloth/internal/core/domain"
)

func completedGarment(id, category, color, season, occasion, material, pattern string) domain.Garment {
	return domain.Garment{
		ID:       id,
		Status:   "completed",
		Category: category,
		Color:    color,
		Season:   season,
		Occasion: occasion,
		Material: material,
		Pattern:  pattern,
	}
}

func TestRecommend_ReturnsTopScoredNotRandom(t *testing.T) {
	engine := NewRecommendationEngine()
	garments := []domain.Garment{
		completedGarment("1", "shirt", "navy", "summer", "casual", "cotton", "solid"),
		completedGarment("2", "shirt", "red", "summer", "casual", "cotton", "solid"),
		completedGarment("3", "pants", "beige", "summer", "casual", "cotton", "solid"),
		completedGarment("4", "pants", "purple", "winter", "formal", "wool", "solid"),
		completedGarment("5", "shoes", "white", "summer", "casual", "leather", "solid"),
		completedGarment("6", "shoes", "brown", "winter", "formal", "leather", "solid"),
	}

	req := domain.RecommendationRequest{UserID: "u1", Limit: 1}
	results := engine.Recommend(req, garments, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 outfit, got %d", len(results))
	}

	best := results[0]
	// Navy + beige + white should beat red + purple + brown on color/season coherence.
	if best.Top.ID != "1" || best.Bottom.ID != "3" || best.Footwear.ID != "5" {
		t.Fatalf("expected harmonious summer casual outfit (1,3,5), got top=%s bottom=%s shoe=%s score=%.2f",
			best.Top.ID, best.Bottom.ID, best.Footwear.ID, best.Score)
	}
	if best.Score <= 0 {
		t.Fatal("expected positive score")
	}
	if best.Reasoning.ColorHarmony == "" {
		t.Fatal("expected color harmony reasoning")
	}
}

func TestRecommend_RespectsAvoidColors(t *testing.T) {
	engine := NewRecommendationEngine()
	garments := []domain.Garment{
		completedGarment("1", "shirt", "red", "summer", "casual", "cotton", "solid"),
		completedGarment("2", "shirt", "blue", "summer", "casual", "cotton", "solid"),
		completedGarment("3", "pants", "beige", "summer", "casual", "cotton", "solid"),
		completedGarment("4", "shoes", "white", "summer", "casual", "leather", "solid"),
	}

	prefs := &domain.UserStylePreferences{
		UserID:      "u1",
		AvoidColors: []string{"red"},
	}

	req := domain.RecommendationRequest{UserID: "u1", Limit: 3}
	results := engine.Recommend(req, garments, prefs)
	for _, r := range results {
		if r.Top.Color == "red" {
			t.Fatal("outfit should not include avoided color red")
		}
	}
}

func TestRecommend_SeasonFilter(t *testing.T) {
	engine := NewRecommendationEngine()
	garments := []domain.Garment{
		completedGarment("1", "shirt", "navy", "summer", "casual", "cotton", "solid"),
		completedGarment("2", "shirt", "gray", "winter", "formal", "wool", "solid"),
		completedGarment("3", "pants", "beige", "summer", "casual", "cotton", "solid"),
		completedGarment("4", "shoes", "white", "summer", "casual", "leather", "solid"),
	}

	req := domain.RecommendationRequest{UserID: "u1", Season: "summer", Limit: 3}
	results := engine.Recommend(req, garments, nil)
	if len(results) == 0 {
		t.Fatal("expected at least one summer outfit")
	}
	for _, r := range results {
		if r.Top.ID == "2" {
			t.Fatal("winter top should be filtered out by season")
		}
	}
}

func TestRecommend_PenalizesMultiplePatterns(t *testing.T) {
	scoreSolid, _ := OutfitPatternScore([]domain.Garment{
		{Pattern: "solid"}, {Pattern: "solid"}, {Pattern: "solid"},
	})
	scoreMulti, _ := OutfitPatternScore([]domain.Garment{
		{Pattern: "striped"}, {Pattern: "floral"}, {Pattern: "solid"},
	})
	if scoreSolid <= scoreMulti {
		t.Fatalf("solid outfit should score higher than multi-pattern: solid=%f multi=%f", scoreSolid, scoreMulti)
	}
}
