package services

import (
	"fmt"
	"strings"
)

// clipColors matches ml_pipeline.py COLOR_MAPPING.
var clipColors = []string{
	"red", "blue", "black", "white", "green", "yellow", "pink", "orange",
	"purple", "brown", "gray", "beige", "navy", "multicolor",
}

var neutralColors = map[string]bool{
	"black": true, "white": true, "gray": true, "beige": true, "navy": true,
}

// complementaryPairs: each color maps to its complement(s).
var complementaryPairs = map[string][]string{
	"red":    {"green"},
	"green":  {"red"},
	"blue":   {"orange"},
	"orange": {"blue"},
	"purple": {"yellow"},
	"yellow": {"purple"},
}

// analogousGroups: colors within the same group harmonize.
var analogousGroups = [][]string{
	{"blue", "navy", "purple"},
	{"red", "orange", "pink"},
	{"green", "beige", "brown"},
	{"yellow", "orange", "beige"},
}

// colorHarmonyMatrix stores precomputed pairwise harmony scores.
var colorHarmonyMatrix map[string]map[string]float64

func init() {
	colorHarmonyMatrix = buildColorHarmonyMatrix()
}

func buildColorHarmonyMatrix() map[string]map[string]float64 {
	matrix := make(map[string]map[string]float64)
	for _, c := range clipColors {
		matrix[c] = make(map[string]float64)
		for _, d := range clipColors {
			matrix[c][d] = pairwiseColorHarmony(c, d)
		}
	}
	return matrix
}

func normalizeColor(color string) string {
	return strings.ToLower(strings.TrimSpace(color))
}

// ColorHarmonyScore returns harmony between two garment colors (0.0–1.0).
func ColorHarmonyScore(a, b string) float64 {
	a = normalizeColor(a)
	b = normalizeColor(b)
	if a == "" || b == "" {
		return 0.6
	}
	if row, ok := colorHarmonyMatrix[a]; ok {
		if score, ok := row[b]; ok {
			return score
		}
	}
	return pairwiseColorHarmony(a, b)
}

func pairwiseColorHarmony(a, b string) float64 {
	if a == b {
		if a == "multicolor" {
			return 0.75
		}
		return 0.8 // monochromatic
	}
	if neutralColors[a] || neutralColors[b] {
		return 1.0
	}
	if a == "multicolor" || b == "multicolor" {
		if neutralColors[a] || neutralColors[b] {
			return 1.0
		}
		return 0.7
	}
	if isComplementary(a, b) {
		return 0.9
	}
	if isAnalogous(a, b) {
		return 0.85
	}
	return 0.3 // clash
}

func isComplementary(a, b string) bool {
	for _, comp := range complementaryPairs[a] {
		if comp == b {
			return true
		}
	}
	return false
}

func isAnalogous(a, b string) bool {
	for _, group := range analogousGroups {
		inA, inB := false, false
		for _, c := range group {
			if c == a {
				inA = true
			}
			if c == b {
				inB = true
			}
		}
		if inA && inB {
			return true
		}
	}
	return false
}

// OutfitColorHarmonyScore averages pairwise harmony across three garments.
func OutfitColorHarmonyScore(topColor, bottomColor, shoeColor string) (float64, string) {
	pairs := []struct{ a, b string }{
		{topColor, bottomColor},
		{topColor, shoeColor},
		{bottomColor, shoeColor},
	}
	var sum float64
	for _, p := range pairs {
		sum += ColorHarmonyScore(p.a, p.b)
	}
	avg := sum / float64(len(pairs))

	top := normalizeColor(topColor)
	bottom := normalizeColor(bottomColor)
	shoe := normalizeColor(shoeColor)

	note := describeColorHarmony(top, bottom, shoe, avg)
	return avg, note
}

func describeColorHarmony(top, bottom, shoe string, avg float64) string {
	if avg >= 0.95 {
		return fmt.Sprintf("neutral palette (%s, %s, %s)", top, bottom, shoe)
	}
	if avg >= 0.85 {
		return fmt.Sprintf("harmonious colors (%s + %s + %s)", top, bottom, shoe)
	}
	if avg >= 0.75 {
		return fmt.Sprintf("balanced combination (%s, %s, %s)", top, bottom, shoe)
	}
	return fmt.Sprintf("bold mix (%s, %s, %s)", top, bottom, shoe)
}
