package config

type Config struct {
	Mysql      *Mysql      `yaml:"mysql"`
	Logger     *Logger     `yaml:"logger"`
	ChatConfig *ChatConfig `yaml:"chatconfig"`
	Embeddings *Embeddings `yaml:"embeddings"`
	Redis      *Redis      `yaml:"redis"`
    Compliance *Compliance `yaml:"compliance"`
}
