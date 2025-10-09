package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OpenAIClient はAzure OpenAI REST APIへのリクエストを管理します。
// このクライアントは、リクエストを実際のAzure OpenAIサービスに転送する
// リバースプロキシをエンドポイントとして設定することを想定しています。
type OpenAIClient struct {
	endpoint                string
	apiKey                  string
	apiVersion              string
	chatDeploymentName      string
	embeddingDeploymentName string
	httpClient              *http.Client
}

// NewOpenAIClient は新しいAzure OpenAIクライアントを作成します。
// endpointには、Azure OpenAIの実際のエンドポイント、またはリクエストを転送するプロキシのURLを設定します。
func NewOpenAIClient(endpoint, apiKey, apiVersion, chatDeploymentName, embeddingDeploymentName, proxyURL string) *OpenAIClient {
	// 注意: このプロジェクトでは、HTTPクライアントのTransport層でのプロキシ設定は行いません。
	// 代わりに、endpoint自体にプロキシのURLを指定する方式を採用しています。
	// これは、使用しているプロキシの特殊な仕様に対応するためです。
	transport := &http.Transport{}
	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxy)
			log.Println("HTTPクライアントにプロキシを設定しました:", proxyURL)
		} else {
			log.Printf("警告: 無効なプロキシURLです。プロキシは使用されません: %v", err)
		}
	}

	return &OpenAIClient{
		endpoint:                endpoint,
		apiKey:                  apiKey,
		apiVersion:              apiVersion,
		chatDeploymentName:      chatDeploymentName,
		embeddingDeploymentName: embeddingDeploymentName,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   60 * time.Second,
		},
	}
}

// --- データ構造定義 ---

// ChatMessage チャットメッセージ
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest チャット補完リクエスト
type ChatCompletionRequest struct {
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float32       `json:"temperature,omitempty"`
	TopP        float32       `json:"top_p,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// ChatCompletionResponse チャット補完レスポンス
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		FinishReason string `json:"finish_reason"`
	}
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	}
}

// EmbeddingRequest Embedding APIリクエスト
type EmbeddingRequest struct {
	Input string `json:"input"`
}

// EmbeddingResponse Embedding APIレスポンス
type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	}
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	}
}

// ErrorResponse エラーレスポンス
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Type    string `json:"type"`
	}
}

// --- メソッド定義 ---

// ChatCompletion チャット補完を実行
func (c *OpenAIClient) ChatCompletion(ctx context.Context, messages []ChatMessage, maxTokens int, temperature float32, topP float32, stream bool) (*ChatCompletionResponse, error) {
	// リクエストURLをエンドポイントとデプロイ名から組み立てます。
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		strings.TrimSuffix(c.endpoint, "/"), c.chatDeploymentName, c.apiVersion)

	request := ChatCompletionRequest{
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		TopP:        topP,
		Stream:      stream,
	}

	var response ChatCompletionResponse
	_, err := c.doRequest(ctx, url, request, &response)
	if err != nil {
		return nil, fmt.Errorf("Azure OpenAI API 呼び出しに失敗: %w", err)
	}
	return &response, nil
}

// CreateEmbedding テキストのベクトル表現を生成
func (c *OpenAIClient) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if c.embeddingDeploymentName == "" {
		return nil, fmt.Errorf("Embedding deployment name が設定されていません")
	}

	// リクエストURLをエンドポイントとデプロイ名から組み立てます。
	url := fmt.Sprintf("%s/openai/deployments/%s/embeddings?api-version=%s",
		strings.TrimSuffix(c.endpoint, "/"), c.embeddingDeploymentName, c.apiVersion)

	request := EmbeddingRequest{
		Input: text,
	}

	var embeddingResp EmbeddingResponse
	_, err := c.doRequest(ctx, url, request, &embeddingResp)
	if err != nil {
		return nil, err
	}

	if len(embeddingResp.Data) == 0 || len(embeddingResp.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("APIから有効なEmbeddingが返されませんでした")
	}

	return embeddingResp.Data[0].Embedding, nil
}

// doRequest はHTTPリクエストの実行と基本的なレスポンス処理を行う共通メソッドです。
func (c *OpenAIClient) doRequest(ctx context.Context, url string, requestData interface{}, responseData interface{}) (interface{}, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("API key が設定されていません")
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("リクエストのJSON化に失敗: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("HTTPリクエストの作成に失敗: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTPリクエストの実行に失敗: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("レスポンスの読み取りに失敗: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			return nil, fmt.Errorf("Azure OpenAI API エラー (status: %d): %s", resp.StatusCode, errorResp.Error.Message)
		}
		return nil, fmt.Errorf("Azure OpenAI API エラー (status: %d): %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, responseData); err != nil {
		return nil, fmt.Errorf("レスポンスのJSON解析に失敗: %w", err)
	}

	return responseData, nil
}

// --- 以下、既存のヘルパー関数 (これらはCreateChatCompletionを呼び出すので変更不要) ---

// AnalyzeWeatherData 気象データの分析
func (c *OpenAIClient) AnalyzeWeatherData(ctx context.Context, weatherData string) (string, error) {
	messages := []ChatMessage{
		{
			Role:    "system",
			Content: "あなたは製造業向けの気象データ分析専門家です。提供された気象データを分析し、製造業の需要予測に役立つ洞察を提供してください。",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("以下の気象データを分析して、製造業の需要予測に役立つ洞察を提供してください：\n\n%s", weatherData),
		},
	}

	response, err := c.ChatCompletion(ctx, messages, 1000, 0.7, 0.95, false)
	if err != nil {
		return "", err
	}

	if len(response.Choices) > 0 {
		return response.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("Azure OpenAI からの応答が空です")
}

// PredictDemand 需要予測
func (c *OpenAIClient) PredictDemand(ctx context.Context, weatherData, historicalData, productCategory string) (string, error) {
	messages := []ChatMessage{
		{
			Role:    "system",
			Content: "あなたは製造業の需要予測専門家です。気象データと過去のデータを分析して、指定された製品カテゴリの需要を予測してください。",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("以下の情報を基に「%s」の需要予測を行ってください：\n\n気象データ：\n%s\n\n過去データ：\n%s\n\n具体的な数値予測とその根拠を含めて回答してください。", productCategory, weatherData, historicalData),
		},
	}

	response, err := c.ChatCompletion(ctx, messages, 1500, 0.6, 0.95, false)
	if err != nil {
		return "", err
	}

	if len(response.Choices) > 0 {
		return response.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("Azure OpenAI からの応答が空です")
}

// ExplainPrediction 予測結果の説明
func (c *OpenAIClient) ExplainPrediction(ctx context.Context, prediction, factors string) (string, error) {
	messages := []ChatMessage{
		{
			Role:    "system",
			Content: "あなたは製造業の需要予測結果を説明する専門家です。予測結果とその影響要因を分析し、分かりやすく説明してください。",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("以下の予測結果について、なぜこのような予測になったのか詳しく説明してください：\n\n予測結果：\n%s\n\n影響要因：\n%s", prediction, factors),
		},
	}

	response, err := c.ChatCompletion(ctx, messages, 1200, 0.7, 0.95, false)
	if err != nil {
		return "", err
	}

	if len(response.Choices) > 0 {
		return response.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("Azure OpenAI からの応答が空です")
}

// GenerateInsights 洞察の生成
func (c *OpenAIClient) GenerateInsights(ctx context.Context, weatherData, historicalData string) (string, error) {
	messages := []ChatMessage{
		{
			Role:    "system",
			Content: "あなたは製造業の需要予測と市場分析の専門家です。気象データと過去のデータを組み合わせて、実用的な洞察を提供してください。",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("以下のデータを基に、製造業の需要予測に役立つ洞察を生成してください：\n\n気象データ：\n%s\n\n過去データ：\n%s", weatherData, historicalData),
		},
	}

	response, err := c.ChatCompletion(ctx, messages, 1500, 0.7, 0.95, false)
	if err != nil {
		return "", err
	}

	if len(response.Choices) > 0 {
		return response.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("Azure OpenAI からの応答が空です")
}
