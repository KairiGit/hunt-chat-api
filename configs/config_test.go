package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// テスト用の環境変数を設定
	testCases := map[string]string{
		"PORT":                         "8080",
		"ENVIRONMENT":                  "test",
		"AZURE_OPENAI_ENDPOINT":        "https://test.openai.azure.com/",
		"AZURE_OPENAI_API_KEY":         "test-key",
		"AZURE_OPENAI_MODEL":           "gpt-4",
		"AZURE_OPENAI_API_VERSION":     "2023-12-01-preview",
		"AZURE_OPENAI_DEPLOYMENT_NAME": "test-deployment",
	}

	// 環境変数を設定
	for key, value := range testCases {
		os.Setenv(key, value)
	}

	// テスト後にクリーンアップ
	defer func() {
		for key := range testCases {
			os.Unsetenv(key)
		}
	}()

	// 設定を読み込み
	cfg := LoadConfig()

	// 検証
	if cfg.Port != "8080" {
		t.Errorf("Expected Port to be '8080', got '%s'", cfg.Port)
	}

	if cfg.Environment != "test" {
		t.Errorf("Expected Environment to be 'test', got '%s'", cfg.Environment)
	}

	if cfg.AzureOpenAIEndpoint != "https://test.openai.azure.com/" {
		t.Errorf("Expected AzureOpenAIEndpoint to be 'https://test.openai.azure.com/', got '%s'", cfg.AzureOpenAIEndpoint)
	}

	if cfg.AzureOpenAIAPIKey != "test-key" {
		t.Errorf("Expected AzureOpenAIAPIKey to be 'test-key', got '%s'", cfg.AzureOpenAIAPIKey)
	}

	if cfg.AzureOpenAIModel != "gpt-4" {
		t.Errorf("Expected AzureOpenAIModel to be 'gpt-4', got '%s'", cfg.AzureOpenAIModel)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	// 環境変数をクリア
	vars := []string{
		"PORT", "ENVIRONMENT", "AZURE_OPENAI_ENDPOINT",
		"AZURE_OPENAI_API_KEY", "AZURE_OPENAI_MODEL",
		"AZURE_OPENAI_API_VERSION", "AZURE_OPENAI_DEPLOYMENT_NAME",
	}

	for _, v := range vars {
		os.Unsetenv(v)
	}

	// 設定を読み込み
	cfg := LoadConfig()

	// デフォルト値の検証
	if cfg.Port != "8080" {
		t.Errorf("Expected default Port to be '8080', got '%s'", cfg.Port)
	}

	if cfg.Environment != "development" {
		t.Errorf("Expected default Environment to be 'development', got '%s'", cfg.Environment)
	}
}
