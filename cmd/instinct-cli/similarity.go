package main

import "math"

type SimilarityFunc func(a, b string) float64

func ngramFreq(s string, n int) map[string]int {
	runes := []rune(s)
	freq := make(map[string]int)
	for i := 0; i <= len(runes)-n; i++ {
		freq[string(runes[i:i+n])]++
	}
	return freq
}

func cosineSim(a, b map[string]int) float64 {
	dot, normA, normB := 0.0, 0.0, 0.0
	for k, va := range a {
		normA += float64(va) * float64(va)
		if vb, ok := b[k]; ok {
			dot += float64(va) * float64(vb)
		}
	}
	for _, vb := range b {
		normB += float64(vb) * float64(vb)
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func bigramSimilarity(a, b string) float64 {
	return cosineSim(ngramFreq(a, 2), ngramFreq(b, 2))
}

func trigramSimilarity(a, b string) float64 {
	return cosineSim(ngramFreq(a, 3), ngramFreq(b, 3))
}

func overlapSimilarity(a, b string) float64 {
	fa, fb := ngramFreq(a, 2), ngramFreq(b, 2)
	intersection, union := 0, 0
	seen := make(map[string]bool)
	for k := range fa {
		seen[k] = true
		if fb[k] > 0 {
			intersection++
		}
		union++
	}
	for k := range fb {
		if !seen[k] {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

type SimilarityScores struct {
	Bigram  float64
	Trigram float64
	Overlap float64
}

func computeAllScores(a, b string) SimilarityScores {
	return SimilarityScores{
		Bigram:  bigramSimilarity(a, b),
		Trigram: trigramSimilarity(a, b),
		Overlap: overlapSimilarity(a, b),
	}
}

func anyAbove(scores SimilarityScores, threshold float64) bool {
	return scores.Bigram >= threshold || scores.Trigram >= threshold || scores.Overlap >= threshold
}

const defaultSimilarityThreshold = 0.15

func similarityThresholdFromConfig(cfg *InstinctConfig) float64 {
	if cfg != nil && cfg.Dedup.SimilarityThreshold > 0 {
		return cfg.Dedup.SimilarityThreshold
	}
	return defaultSimilarityThreshold
}
