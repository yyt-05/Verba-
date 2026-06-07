package config

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Port                 string
	SiliconFlowAPIKey    string
	SiliconFlowBaseURL   string
	ASRModel             string
	TranslateModel       string
	DiarizationProvider  string
	DeepgramAPIKey       string
	DeepgramRealtimeURL  string
	DeepgramModel        string
	DeepgramLanguage     string
	TTSProvider          string
	DashScopeAPIKey      string
	DashScopeRealtimeURL string
	TTSModel             string
	TTSVoice             string
	TTSLanguage          string
	MaxSessionMin        int
	BudgetCapUSD         float64
}

func Load() *Config {
	loadDotEnv()

	return &Config{
		Port:                 envOrDefault("VERBA_PORT", "8080"),
		SiliconFlowAPIKey:    os.Getenv("SILICONFLOW_API_KEY"),
		SiliconFlowBaseURL:   envOrDefault("SILICONFLOW_BASE_URL", "https://api.siliconflow.cn/v1"),
		ASRModel:             envOrDefault("VERBA_ASR_MODEL", "FunAudioLLM/SenseVoiceSmall"),
		TranslateModel:       envOrDefault("VERBA_TRANSLATE_MODEL", "deepseek-ai/DeepSeek-V3"),
		DiarizationProvider:  envOrDefault("VERBA_DIARIZATION_PROVIDER", ""),
		DeepgramAPIKey:       os.Getenv("DEEPGRAM_API_KEY"),
		DeepgramRealtimeURL:  envOrDefault("DEEPGRAM_REALTIME_URL", "wss://api.deepgram.com/v1/listen"),
		DeepgramModel:        envOrDefault("DEEPGRAM_MODEL", "nova-3"),
		DeepgramLanguage:     envOrDefault("DEEPGRAM_LANGUAGE", "en"),
		TTSProvider:          envOrDefault("VERBA_TTS_PROVIDER", "dashscope"),
		DashScopeAPIKey:      os.Getenv("DASHSCOPE_API_KEY"),
		DashScopeRealtimeURL: envOrDefault("DASHSCOPE_REALTIME_URL", "wss://dashscope.aliyuncs.com/api-ws/v1/realtime"),
		TTSModel:             envOrDefault("VERBA_TTS_MODEL", "qwen3-tts-flash-realtime"),
		TTSVoice:             envOrDefault("VERBA_TTS_VOICE", "Cherry"),
		TTSLanguage:          envOrDefault("VERBA_TTS_LANGUAGE", "Chinese"),
		MaxSessionMin:        60,
		BudgetCapUSD:         1.0,
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
