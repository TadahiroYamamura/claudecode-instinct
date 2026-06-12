package main

import (
	"math"
	"testing"
)

func approxEqual(a, b, eps float64) bool {
	return math.Abs(a-b) < eps
}

// bigramSimilarityは完全一致で1.0を返す
func TestBigramSimilarity_IdenticalReturnsOne(t *testing.T) {
	s := "テスト前にlintを通す"
	if got := bigramSimilarity(s, s); !approxEqual(got, 1.0, 1e-9) {
		t.Errorf("expected 1.0, got %f", got)
	}
}

// bigramSimilarityは共通部分なしで0.0を返す
func TestBigramSimilarity_NoOverlapReturnsZero(t *testing.T) {
	if got := bigramSimilarity("あいうえお", "ABCDEFG"); got != 0.0 {
		t.Errorf("expected 0.0, got %f", got)
	}
}

// bigramSimilarityは類似テキストで0より大きく1未満を返す
func TestBigramSimilarity_SimilarTextReturnsIntermediate(t *testing.T) {
	a := "テスト前にlintを通す"
	b := "lintエラーを解消してからテストを走らせる"
	got := bigramSimilarity(a, b)
	if got <= 0.0 || got >= 1.0 {
		t.Errorf("expected 0 < sim < 1, got %f", got)
	}
}

// trigramSimilarityは完全一致で1.0を返す
func TestTrigramSimilarity_IdenticalReturnsOne(t *testing.T) {
	s := "テスト前にlintを通す"
	if got := trigramSimilarity(s, s); !approxEqual(got, 1.0, 1e-9) {
		t.Errorf("expected 1.0, got %f", got)
	}
}

// overlapSimilarityは完全一致で1.0を返す
func TestOverlapSimilarity_IdenticalReturnsOne(t *testing.T) {
	s := "テスト前にlintを通す"
	if got := overlapSimilarity(s, s); !approxEqual(got, 1.0, 1e-9) {
		t.Errorf("expected 1.0, got %f", got)
	}
}

// overlapSimilarityは共通部分なしで0.0を返す
func TestOverlapSimilarity_NoOverlapReturnsZero(t *testing.T) {
	if got := overlapSimilarity("あいうえお", "ABCDEFG"); got != 0.0 {
		t.Errorf("expected 0.0, got %f", got)
	}
}
