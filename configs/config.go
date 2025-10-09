package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	Port                               string
	AzureOpenAIEndpoint                string
	AzureOpenAIAPIKey                  string
	AzureOpenAIModel                   string
	AzureOpenAIAPIVersion              string
	AzureOpenAIChatDeploymentName      string
	AzureOpenAIEmbeddingDeploymentName string
	Environment                        string
	OpenWeatherMapAPIKey               string
	QdrantURL                          string
	QdrantAPIKey                       string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Port:                               getEnv("PORT", "8080"),
		AzureOpenAIEndpoint:                getEnv("AZURE_OPENAI_ENDPOINT", ""),
		AzureOpenAIAPIKey:                  getEnv("AZURE_OPENAI_API_KEY", ""),
		AzureOpenAIModel:                   getEnv("AZURE_OPENAI_MODEL", "gpt-4o-mini"),
		AzureOpenAIAPIVersion:              getEnv("AZURE_OPENAI_API_VERSION", "2023-12-01-preview"),
		AzureOpenAIChatDeploymentName:      getEnv("AZURE_OPENAI_CHAT_DEPLOYMENT_NAME", "gpt-4o-mini"),
		AzureOpenAIEmbeddingDeploymentName: getEnv("AZURE_OPENAI_EMBEDDING_DEPLOYMENT_NAME", "text-embedding-3-small"),
		Environment:                        getEnv("ENVIRONMENT", "development"),
		OpenWeatherMapAPIKey:               getEnv("OPENWEATHERMAP_API_KEY", ""),
		QdrantURL:                          getEnv("QDRANT_URL", "127.0.0.1:6334"),
		QdrantAPIKey:                       getEnv("QDRANT_API_KEY", ""),
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
