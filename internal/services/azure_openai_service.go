package services

import (
	"context"
	"fmt"
	"time"

	"hunt-chat-api/pkg/azure"
)

// AzureOpenAIService Azure OpenAI API サービス
type AzureOpenAIService struct {
	client *azure.OpenAIClient
}

// NewAzureOpenAIService 新しいAzure OpenAI サービスを作成
func NewAzureOpenAIService(endpoint, apiKey, apiVersion, deploymentName string) *AzureOpenAIService {
	client := azure.NewOpenAIClient(endpoint, apiKey, apiVersion, deploymentName)
	return &AzureOpenAIService{
		client: client,
	}
}

// ChatMessage チャットメッセージ構造体（互換性のため）
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse Azure OpenAI チャットレスポンス構造体（互換性のため）
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// CreateChatCompletion Azure OpenAI チャット補完を作成
func (aos *AzureOpenAIService) CreateChatCompletion(messages []ChatMessage, maxTokens int, temperature float32) (*ChatResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ChatMessageをazure.ChatMessageに変換
	azureMessages := make([]azure.ChatMessage, len(messages))
	for i, msg := range messages {
		azureMessages[i] = azure.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Azure OpenAI REST API を呼び出し
	response, err := aos.client.ChatCompletion(ctx, azureMessages, maxTokens, temperature, 0.95, false)
	if err != nil {
		return nil, fmt.Errorf("Azure OpenAI API 呼び出しに失敗: %w", err)
	}

	// レスポンスを互換性のある形式に変換
	chatResponse := &ChatResponse{
		ID:      response.ID,
		Object:  response.Object,
		Created: response.Created,
		Model:   response.Model,
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		},
	}

	// Choicesを変換
	chatResponse.Choices = make([]struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	}, len(response.Choices))

	for i, choice := range response.Choices {
		chatResponse.Choices[i].Index = choice.Index
		chatResponse.Choices[i].Message.Role = choice.Message.Role
		chatResponse.Choices[i].Message.Content = choice.Message.Content
		chatResponse.Choices[i].FinishReason = choice.FinishReason
	}

	return chatResponse, nil
}

// AnalyzeWeatherData 気象データを分析
func (aos *AzureOpenAIService) AnalyzeWeatherData(weatherData string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return aos.client.AnalyzeWeatherData(ctx, weatherData)
}

// GenerateDemandInsights 需要予測の洞察を生成
func (aos *AzureOpenAIService) GenerateDemandInsights(weatherData, historicalData string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return aos.client.GenerateInsights(ctx, weatherData, historicalData)
}

// PredictDemandWithAI AIを使用した需要予測
func (aos *AzureOpenAIService) PredictDemandWithAI(weatherData, historicalData, productCategory string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return aos.client.PredictDemand(ctx, weatherData, historicalData, productCategory)
}

// ExplainForecast 予測結果の説明可能性を提供
func (aos *AzureOpenAIService) ExplainForecast(forecastData, factors string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return aos.client.ExplainPrediction(ctx, forecastData, factors)
}
