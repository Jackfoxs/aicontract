package config

import "testing"

func TestComplianceApplyDefaults(t *testing.T) {
	c := &Compliance{}
	c.ApplyDefaults()

	if c.WorkerConcurrency != 2 {
		t.Fatalf("expected WorkerConcurrency default 2, got %d", c.WorkerConcurrency)
	}
	if c.QueueKey != "compliance:jobs" {
		t.Fatalf("unexpected QueueKey default: %s", c.QueueKey)
	}
	if c.RetryLimit != 3 {
		t.Fatalf("expected RetryLimit default 3, got %d", c.RetryLimit)
	}
	if c.TaskTimeout <= 0 {
		t.Fatalf("expected positive TaskTimeout, got %v", c.TaskTimeout)
	}
	if c.AIThresholdDefault <= 0 {
		t.Fatalf("expected positive AIThresholdDefault, got %v", c.AIThresholdDefault)
	}
	if c.ReportOutputDir != "uploads/compliance" {
		t.Fatalf("unexpected ReportOutputDir default: %s", c.ReportOutputDir)
	}
	if !c.SemanticMatchEnabled() {
		t.Fatalf("expected SemanticMatchEnabled default true")
	}
}

func TestComplianceSemanticMatchToggle(t *testing.T) {
	disabled := false
	c := &Compliance{EnableSemanticMatch: &disabled}
	if c.SemanticMatchEnabled() {
		t.Fatalf("expected semantic match disabled")
	}
}
