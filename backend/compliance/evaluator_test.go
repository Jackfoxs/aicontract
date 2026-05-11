package compliance

import (
	"testing"

	"backend/models"
)

func TestSemanticEvaluatorEvaluate(t *testing.T) {
	evaluator := NewSemanticEvaluator()
	matches := []RuleMatch{
		{
			Rule:            &models.DocumentChunk{ID: 1, Content: "质保承诺"},
			ResponseHit:     true,
			ResponseExcerpt: "供应商提供质保承诺",
		},
		{
			Rule:          &models.DocumentChunk{ID: 2, Content: "交货期"},
			TenderHit:     true,
			TenderExcerpt: "交货期为30天",
		},
	}

	results := evaluator.Evaluate(matches, "供应商提供质保承诺。", 0.7)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Status != "matched" {
		t.Fatalf("expected first result matched, got %s", results[0].Status)
	}
	if results[1].Status != "inconsistent" {
		t.Fatalf("expected second result inconsistent, got %s", results[1].Status)
	}
}
