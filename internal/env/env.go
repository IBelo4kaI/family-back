package env

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Env struct{}

// Init подхватывает .env, если он есть; в проде переменные приходят из окружения.
func (e *Env) Init() {
	_ = godotenv.Load()
}

func (e *Env) GetAddr() string {
	if v := os.Getenv("ADDR"); v != "" {
		return v
	}
	return "0.0.0.0:8080"
}

func (e *Env) GetDbString() string {
	return os.Getenv("DB_STRING")
}

func (e *Env) GetJWTSecret() string {
	return os.Getenv("JWT_SECRET")
}

func (e *Env) GetCORSOrigins() []string {
	var origins []string
	for _, o := range strings.Split(os.Getenv("CORS_ORIGINS"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}

// GetProverkachekaTokens — токены сервиса «Проверка чека онлайн» через запятую;
// у каждого свой лимит запросов, при исчерпании берётся следующий.
func (e *Env) GetProverkachekaTokens() []string {
	var tokens []string
	for _, t := range strings.Split(os.Getenv("PROVERKACHEKA_TOKENS"), ",") {
		if t = strings.TrimSpace(t); t != "" {
			tokens = append(tokens, t)
		}
	}
	return tokens
}
