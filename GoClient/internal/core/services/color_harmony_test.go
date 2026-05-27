package services

import "testing"

func TestColorHarmonyScore_NeutralPairsHigh(t *testing.T) {
	score := ColorHarmonyScore("black", "white")
	if score < 0.99 {
		t.Fatalf("expected neutral pair score ~1.0, got %f", score)
	}
}

func TestColorHarmonyScore_Complementary(t *testing.T) {
	score := ColorHarmonyScore("red", "green")
	if score < 0.85 || score > 0.95 {
		t.Fatalf("expected complementary score ~0.9, got %f", score)
	}
}

func TestColorHarmonyScore_Monochromatic(t *testing.T) {
	score := ColorHarmonyScore("blue", "blue")
	if score < 0.75 || score > 0.85 {
		t.Fatalf("expected monochromatic score ~0.8, got %f", score)
	}
}

func TestColorHarmonyScore_Clash(t *testing.T) {
	score := ColorHarmonyScore("red", "purple")
	if score > 0.5 {
		t.Fatalf("expected clash score low, got %f", score)
	}
}

func TestOutfitColorHarmonyScore_AveragesPairs(t *testing.T) {
	score, note := OutfitColorHarmonyScore("navy", "beige", "white")
	if score < 0.9 {
		t.Fatalf("expected high harmony for neutrals, got %f", score)
	}
	if note == "" {
		t.Fatal("expected non-empty harmony note")
	}
}
