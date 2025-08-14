package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	Port                      string
	AzureOpenAIEndpoint       string
	AzureOpenAIAPIKey         string
	AzureOpenAIModel          string
	AzureOpenAIAPIVersion     string
	AzureOpenAIDeploymentName string
	Environment               string
	OpenWeatherMapAPIKey      string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Port:                      getEnv("PORT", "8080"),
		AzureOpenAIEndpoint:       getEnv("AZURE_OPENAI_ENDPOINT", ""),
		AzureOpenAIAPIKey:         getEnv("AZURE_OPENAI_API_KEY", ""),
		AzureOpenAIModel:          getEnv("AZURE_OPENAI_MODEL", "gpt-4o-mini"),
		AzureOpenAIAPIVersion:     getEnv("AZURE_OPENAI_API_VERSION", "2023-12-01-preview"),
		AzureOpenAIDeploymentName: getEnv("AZURE_OPENAI_DEPLOYMENT_NAME", "gpt-4o-mini"),
		Environment:               getEnv("ENVIRONMENT", "development"),
		OpenWeatherMapAPIKey:      getEnv("OPENWEATHERMAP_API_KEY", ""),
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
