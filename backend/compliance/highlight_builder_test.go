package compliance

import (
	"context"
	"testing"

	"backend/models"

	_ "github.com/glebarez/sqlite"
	"xorm.io/xorm"
)

func TestHighlightBuilderCreateHighlights(t *testing.T) {
	db, err := xorm.NewEngine("sqlite", "file:highlight_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	defer db.Close()
	if err := db.Sync2(models.GetModels()...); err != nil {
		t.Fatalf("sync models: %v", err)
	}

	b := NewHighlightBuilder(db)
	issues := []IssueResult{
		{
			Rule:            &models.DocumentChunk{ID: 1, Content: "质保承诺"},
			Status:          "matched",
			ResponseExcerpt: "供应商提供质保承诺",
			TenderExcerpt:   "本项目要求质保承诺",
		},
	}

	pairs, err := b.CreateHighlights(context.Background(), 1001, issues, "本项目要求质保承诺", "供应商提供质保承诺")
	if err != nil {
		t.Fatalf("CreateHighlights failed: %v", err)
	}
	pair, ok := pairs[1]
	if !ok {
		t.Fatalf("expected highlight pair for rule")
	}
	if pair.ResponseID == 0 || pair.TenderID == 0 {
		t.Fatalf("expected both highlights generated")
	}
}
