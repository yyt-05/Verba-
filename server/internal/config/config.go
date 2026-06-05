package config

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Port               string
	SiliconFlowAPIKey  string
	SiliconFlowBaseURL string
	ASRModel           string
	TranslateModel     string
	MaxSessionMin      int
	BudgetCapUSD       float64
}

func Load() *Config {
	loadDotEnv()

	return &Config{
		Port:               envOrDefault("VERBA_PORT", "8080"),
		SiliconFlowAPIKey:  os.Getenv("SILICONFLOW_API_KEY"),
		SiliconFlowBaseURL: envOrDefault("SILICONFLOW_BASE_URL", "https://api.siliconflow.cn/v1"),
		ASRModel:           envOrDefault("VERBA_ASR_MODEL", "FunAudioLLM/SenseVoiceSmall"),
		TranslateModel:     envOrDefault("VERBA_TRANSLATE_MODEL", "deepseek-ai/DeepSeek-V3"),
		MaxSessionMin:      60,
		BudgetCapUSD:       1.0,
	}
}

// loadDotEnv reads server/.env and sets environment variables.
// Skips if .env doesn't exist. Never overwrites existing env vars.
func loadDotEnv() {
	// Look for .env in server/ directory
	envPath := filepath.Join(".", ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		// Also try from working directory
		envPath = ".env"
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			return
		}
	}

	f, err := os.Open(envPath)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Only set if not already in environment
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}

	log.Printf("[config] loaded .env from %s", envPath)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
