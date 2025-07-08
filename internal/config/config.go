package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Vector    VectorConfig
	Embedding EmbeddingConfig
	Log       LogConfig
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	URL string
}

type VectorConfig struct {
	Provider string
	Qdrant   QdrantConfig
	// Future providers can be added here
	// Pinecone PineconeConfig
	// Weaviate WeaviateConfig
}

type QdrantConfig struct {
	Host       string
	Port       int
	Collection string
	APIKey     string
	UseTLS     bool
}

type EmbeddingConfig struct {
	Provider string
	OpenAI   OpenAIConfig
	Gemini   GeminiConfig
	Compatible OpenAICompatibleConfig
}

type OpenAIConfig struct {
	APIKey string
	Model  string
}

type GeminiConfig struct {
	APIKey string
	Model  string
}

type OpenAICompatibleConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

type LogConfig struct {
	Level string
}

func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		logrus.Debug("No .env file found")
	}

	config := &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "postgres://mentis:mentis@localhost:5432/mentis?sslmode=disable"),
		},
		Vector: VectorConfig{
			Provider: getEnv("VECTOR_PROVIDER", "qdrant"),
			Qdrant: QdrantConfig{
				Host:       getEnv("QDRANT_HOST", "localhost"),
				Port:       getEnvInt("QDRANT_PORT", 6334),
				Collection: getEnv("QDRANT_COLLECTION", "mentis"),
				APIKey:     getEnv("QDRANT_API_KEY", ""),
				UseTLS:     getEnvBool("QDRANT_USE_TLS", false),
			},
		},
		Embedding: EmbeddingConfig{
			Provider: getEnv("EMBEDDING_PROVIDER", "mock"),
			OpenAI: OpenAIConfig{
				APIKey: getEnv("OPENAI_API_KEY", ""),
				Model:  getEnv("OPENAI_MODEL", "text-embedding-3-small"),
			},
			Gemini: GeminiConfig{
				APIKey: getEnv("GEMINI_API_KEY", ""),
				Model:  getEnv("GEMINI_MODEL", "text-embedding-004"),
			},
			Compatible: OpenAICompatibleConfig{
				BaseURL: getEnv("EMBEDDING_BASE_URL", "http://localhost:11434/v1"),
				APIKey:  getEnv("EMBEDDING_API_KEY", ""),
				Model:   getEnv("EMBEDDING_MODEL", "nomic-embed-text"),
			},
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func SetupLogging(level string) {
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}

	logrus.SetLevel(logLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z",
	})
}