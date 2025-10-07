package services

import (
	"context"
	"fmt"
	"time"

	"hunt-chat-api/internal/models"
	"hunt-chat-api/pkg/azure"
)

// AzureOpenAIService Azure OpenAI API サービス
type AzureOpenAIService struct {
	client *azure.OpenAIClient
}

// NewAzureOpenAIService 新しいAzure OpenAI サービスを作成
func NewAzureOpenAIService(endpoint, apiKey, apiVersion, deploymentName, proxyURL string) *AzureOpenAIService {
	client := azure.NewOpenAIClient(endpoint, apiKey, apiVersion, deploymentName, proxyURL)
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
	}, len(response.Choices)) // Corrected: response.Choices instead of response.Choices

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

// GenerateQuestionFromAnomaly 異常データから質問を生成する
func (aos *AzureOpenAIService) GenerateQuestionFromAnomaly(anomaly models.Anomaly) (string, error) {
	// プロンプトの構築
	prompt := fmt.Sprintf(
		"あなたは優秀な需要予測コンサルタントです。以下の異常データについて、担当者が原因を特定しやすくなるような、自然で具体的な質問を日本語で1つだけ生成してください。質問以外の余計な言葉は含めないでください。\n\n## 異常データ\n- **発生日**: %s\n- **製品**: %s\n- **事象**: %s\n\n## 質問の例\n- 「この日は何か特別な販促活動やイベントがありましたか？」\n- 「この時期の競合他社の動きで、何か特筆すべきことはありましたか？」\n- 「この日の天候は、過去のデータと比較してどの程度珍しいものだったのでしょうか？」",
		anomaly.Date,
		anomaly.ProductID,
		anomaly.Description,
	)

	messages := []ChatMessage{
		{Role: "system", Content: "あなたは、需要予測の専門家として、データから読み取れる異常について質問を生成するAIです。"},
		{Role: "user", Content: prompt},
	}

	// Azure OpenAI にリクエストを送信
	resp, err := aos.CreateChatCompletion(messages, 150, 0.7)
	if err != nil {
		return "", fmt.Errorf("AIからの質問生成に失敗しました: %w", err)
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("AIから有効な回答が得られませんでした")
}

// ProcessChatWithData は、チャットメッセージとオプションのデータを受け取り、AIで処理します。
func (aos *AzureOpenAIService) ProcessChatWithData(chatMessage string, data string) (string, error) {
	// システムプロンプトを定義
	systemPrompt := "あなたは、需要予測の専門家アシスタントです。ユーザーから提供された販売実績データ、気象データ、および定性的な情報（経験や勘）を統合的に分析し、需要予測に関する質問に答えたり、新しい予測を生成したりします。"

	// ユーザープロンプトを構築
	userPrompt := fmt.Sprintf("以下の情報を考慮して、回答してください。\n\n## ユーザーからのメッセージ\n%s\n", chatMessage)

	if data != "" {
		userPrompt += fmt.Sprintf("\n## アップロードされたデータ (CSV形式)\n%s\n", data)
	}

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// Azure OpenAI にリクエストを送信
	resp, err := aos.CreateChatCompletion(messages, 1000, 0.7)
	if err != nil {
		return "", fmt.Errorf("AI処理中にエラーが発生しました: %w", err)
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("AIから有効な回答が得られませんでした")
}
