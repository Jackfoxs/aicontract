package config

import "time"

// Compliance 定义合规模块配置
type Compliance struct {
	WorkerConcurrency   int           `yaml:"worker_concurrency"`
	QueueKey            string        `yaml:"queue_key"`
	RetryLimit          int           `yaml:"retry_limit"`
	TaskTimeout         time.Duration `yaml:"task_timeout"`
	AIThresholdDefault  float64       `yaml:"ai_threshold_default"`
	ReportOutputDir     string        `yaml:"report_output_dir"`
	EnableSemanticMatch *bool         `yaml:"enable_semantic_match"`
	LLMBatchSize        int           `yaml:"llm_batch_size"`
	LLMConcurrency      int           `yaml:"llm_concurrency"`
}

// ApplyDefaults 填充缺省配置
func (c *Compliance) ApplyDefaults() {
	if c.WorkerConcurrency <= 0 {
		c.WorkerConcurrency = 2
	}
	if c.QueueKey == "" {
		c.QueueKey = "compliance:jobs"
	}
	if c.RetryLimit <= 0 {
		c.RetryLimit = 3
	}
	if c.TaskTimeout <= 0 {
		c.TaskTimeout = 10 * time.Minute
	}
	if c.AIThresholdDefault <= 0 {
		c.AIThresholdDefault = 0.75
	}
	if c.ReportOutputDir == "" {
		c.ReportOutputDir = "uploads/compliance"
	}
	if c.EnableSemanticMatch == nil {
		defaultVal := true
		c.EnableSemanticMatch = &defaultVal
	}
	if c.LLMBatchSize <= 0 {
		c.LLMBatchSize = 5
	}
	if c.LLMConcurrency <= 0 {
		c.LLMConcurrency = 3
	}
}

// SemanticMatchEnabled 是否启用语义匹配
func (c *Compliance) SemanticMatchEnabled() bool {
	if c.EnableSemanticMatch == nil {
		return true
	}
	return *c.EnableSemanticMatch
}
