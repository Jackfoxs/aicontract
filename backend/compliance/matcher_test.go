package compliance

import (
	"context"
	"testing"

	"backend/models"
)

func TestRuleMatcherMatchRules(t *testing.T) {
	m := NewRuleMatcher(nil)
	rules := []*models.DocumentChunk{
		{ID: 1, Title: "要求A", Content: "质保承诺"},
		{ID: 2, Title: "要求B", Content: "交货期"},
	}

	matches := m.MatchRules(context.Background(), rules, "本项目质保承诺包括...交货期为30天", "供应商提供质保承诺与售后服务")

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	if !matches[0].ResponseHit {
		t.Fatalf("expected first rule to match response")
	}
	if matches[1].ResponseHit {
		t.Fatalf("second rule should not match response")
	}
}
