package config

import "time"

// ChatConfig 聊天配置
type ChatConfig struct {
	ModelType      string        `yaml:"model_type"`
	OwnerAPIKey    string        `yaml:"owner_apikey"`
	BaseURL        string        `yaml:"base_url"`
	SystemDefault  string        `yaml:"system_default"`
	UserDefault    string        `yaml:"user_default"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
}

// ApplyDefaults 补齐默认值
func (c *ChatConfig) ApplyDefaults() {
	if c == nil {
		return
	}
	if c.RequestTimeout <= 0 {
		c.RequestTimeout = 60 * time.Second
	}
}

// Embeddings 嵌入模型配置
type Embeddings struct {
	APIKey    string `yaml:"apikey"`
	Endpoint  string `yaml:"endpoint"`
	Embedding string `yaml:"embedding"`
}
