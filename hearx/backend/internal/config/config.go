package config

import "os"

type Config struct {
	Port           string
	ASRProvider    string
	LLMProvider    string
	OpenAIKey      string
	AnthropicKey   string
	DeepgramKey    string
}

func Load() *Config {
	return &Config{
		Port:         getEnv("PORT", "8080"),
		ASRProvider:  getEnv("ASR_PROVIDER", "whisper"),
		LLMProvider:  getEnv("LLM_PROVIDER", "openai"),
		OpenAIKey:    os.Getenv("OPENAI_API_KEY"),
		AnthropicKey: os.Getenv("ANTHROPIC_API_KEY"),
		DeepgramKey:  os.Getenv("DEEPGRAM_API_KEY"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
