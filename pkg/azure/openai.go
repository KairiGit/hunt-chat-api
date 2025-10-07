package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIClient Azure OpenAI REST API クライアント
type OpenAIClient struct {
	endpoint   string
	apiKey     string
	apiVersion string
	deployment string
	proxyURL   string
	httpClient *http.Client
}

// NewOpenAIClient 新しいAzure OpenAI クライアントを作成
func NewOpenAIClient(endpoint, apiKey, apiVersion, deployment, proxyURL string) *OpenAIClient {
	return &OpenAIClient{
		endpoint:   endpoint,
		apiKey:     apiKey,
		apiVersion: apiVersion,
		deployment: deployment,
		proxyURL:   proxyURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

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
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// ErrorResponse エラーレスポンス
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// ChatCompletion チャット補完を実行
func (c *OpenAIClient) ChatCompletion(ctx context.Context, messages []ChatMessage, maxTokens int, temperature float32, topP float32, stream bool) (*ChatCompletionResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("API key が設定されていません")
	}

	// リクエストURLを決定
	var url string
	if c.proxyURL != "" {
		// プロキシURLが設定されていれば、それを最優先で使用
		url = c.proxyURL
	} else {
		// 通常のAzure OpenAI URLを組み立て
		url = fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
			strings.TrimSuffix(c.endpoint, "/"), c.deployment, c.apiVersion)
	}

	// リクエストボディの作成
	request := ChatCompletionRequest{
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		TopP:        topP,
		Stream:      stream,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("リクエストのJSON化に失敗: %w", err)
	}

	// HTTPリクエストの作成
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("HTTPリクエストの作成に失敗: %w", err)
	}

	// ヘッダーの設定
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.apiKey)

	// デバッグ情報をログ出力
	fmt.Printf("DEBUG: リクエストURL: %s\n", url)
	fmt.Printf("DEBUG: APIキー: %s...\n", c.apiKey[:10])
	fmt.Printf("DEBUG: リクエストボディ: %s\n", string(requestBody))

	// リクエストの実行
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTPリクエストの実行に失敗: %w", err)
	}
	defer resp.Body.Close()

	// レスポンスの読み取り
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("レスポンスの読み取りに失敗: %w", err)
	}

	// エラーハンドリング
	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("Azure OpenAI API エラー (status: %d): %s", resp.StatusCode, errorResp.Error.Message)
		}
		return nil, fmt.Errorf("Azure OpenAI API エラー (status: %d): %s", resp.StatusCode, string(body))
	}

	// レスポンスのデコード
	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("レスポンスのJSON解析に失敗: %w", err)
	}

	return &chatResp, nil
}

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
