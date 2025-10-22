package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	config "hunt-chat-api/configs"
	"hunt-chat-api/pkg/azure"
	"hunt-chat-api/pkg/models"
)

// AzureOpenAIService Azure OpenAI API サービス
type AzureOpenAIService struct {
	client *azure.OpenAIClient
}

// NewAzureOpenAIService 新しいAzure OpenAI サービスを作成
func NewAzureOpenAIService(endpoint, apiKey, apiVersion, chatDeploymentName, embeddingDeploymentName string) *AzureOpenAIService {
	client := azure.NewOpenAIClient(endpoint, apiKey, apiVersion, chatDeploymentName, embeddingDeploymentName, "") // proxyURLは不要になったため空文字列を渡す
	return &AzureOpenAIService{
		client: client,
	}
}

// ChatMessage チャットメッセージ構造体（互換性のため）
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnomalyQuestionWithChoices AIが生成した質問と選択肢を格納する構造体
type AnomalyQuestionWithChoices struct {
	Question string   `json:"question"`
	Choices  []string `json:"choices"`
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

// GenerateQuestionAndChoicesFromAnomaly は、異常データから質問と回答の選択肢を生成します。
func (aos *AzureOpenAIService) GenerateQuestionAndChoicesFromAnomaly(anomaly models.Anomaly) (*AnomalyQuestionWithChoices, error) {
	prompt := fmt.Sprintf(
		`あなたは優秀な需要予測コンサルタントです。以下の売上異常データについて、担当者が原因を特定しやすくなるような、自然で具体的な質問と、考えられる原因の選択肢を生成してください。
		レスポンスは必ず以下のJSON形式で返してください。

		# 異常データ
		- 発生日: %s
		- 製品: %s
		- 事象: %s

		# 出力形式 (JSON)
		{
		  "question": "（ユーザーへの自然な質問文）",
		  "choices": [
		    "キャンペーン・販促活動",
		    "天候の影響",
		    "競合他社の動き",
		    "特に思い当たる節はない",
		    "その他（自由記述）"
		  ]
		}`,
		anomaly.Date,
		anomaly.ProductID,
		anomaly.Description,
	)

	messages := []ChatMessage{
		{Role: "system", Content: "あなたは、JSON形式で応答するAIアシスタントです。"},
		{Role: "user", Content: prompt},
	}

	resp, err := aos.CreateChatCompletion(messages, 300, 0.5)
	if err != nil {
		return nil, fmt.Errorf("AIからの質問生成に失敗しました: %w", err)
	}

	if len(resp.Choices) > 0 {
		var result AnomalyQuestionWithChoices
		// AIの出力からJSON部分を抽出する（```json ... ``` のようなマークダウン形式を考慮）
		jsonString := resp.Choices[0].Message.Content
		if strings.HasPrefix(jsonString, "```json") {
			jsonString = strings.TrimPrefix(jsonString, "```json")
			jsonString = strings.TrimSuffix(jsonString, "```")
		}

		if err := json.Unmarshal([]byte(jsonString), &result); err != nil {
			return nil, fmt.Errorf("AIの応答JSONの解析に失敗しました: %w. Response: %s", err, jsonString)
		}
		return &result, nil
	}

	return nil, fmt.Errorf("AIから有効な回答が得られませんでした")
}

// ProcessChatWithContext は、チャットメッセージと事前の分析コンテキストを受け取り、AIで処理します。
func (aos *AzureOpenAIService) ProcessChatWithContext(chatMessage string, context string) (string, error) {
	// システムプロンプトを定義
	systemPrompt := "あなたは、需要予測の専門家アシスタントです。ユーザーから提供された分析コンテキスト（ファイル概要、統計、データサンプル）と、追加の定性的な情報（経験や勘）を統合的に分析し、需要予測に関する質問に答えてください。"

	// ユーザープロンプトを構築
	userPrompt := fmt.Sprintf("以下の情報を考慮して、回答してください。\n\n## ユーザーからのメッセージ\n%s\n", chatMessage)

	if context != "" {
		userPrompt += fmt.Sprintf("\n## 事前分析コンテキスト\n%s\n", context)
	}

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// Azure OpenAI にリクエストを送信
	resp, err := aos.CreateChatCompletion(messages, 2000, 0.7)
	if err != nil {
		return "", fmt.Errorf("AI処理中にエラーが発生しました: %w", err)
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("AIから有効な回答が得られませんでした")
}

// CreateEmbedding はテキストのベクトル表現を生成します。
func (aos *AzureOpenAIService) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return aos.client.CreateEmbedding(ctx, text)
}

// ProcessChatWithHistory は、過去のチャット履歴を活用してより良い回答を生成します。
func (aos *AzureOpenAIService) ProcessChatWithHistory(chatMessage string, context string, relevantHistory []string) (string, error) {
	// システムプロンプトをYAMLファイルから読み込み
	promptConfig, err := config.LoadSystemPrompt()
	if err != nil {
		log.Printf("Warning: Failed to load system prompt from YAML, using fallback: %v", err)
		// フォールバックとして簡易的なプロンプトを使用
		systemPrompt := "あなたは、需要予測システム「HUNT」の専門家アシスタントです。検索されたコンテキストがある場合は、必ずそれを最優先で使用し、システム固有の情報を説明してください。一般論は避けてください。"

		// ユーザープロンプトを構築
		userPrompt := fmt.Sprintf("## ユーザーからの質問\n%s\n", chatMessage)

		// 現在のコンテキストを最優先で追加（RAG検索結果）
		if context != "" {
			userPrompt += fmt.Sprintf("\n## 🔍 検索されたコンテキスト（必ずこれを基に回答してください）\n%s\n", context)
			userPrompt += "\n**重要**: 上記の検索結果に含まれる具体的な情報を使用して回答してください。一般論ではなく、このシステム固有の情報を説明してください。\n"
			userPrompt += "\n**出典の明記**: 回答する際は、以下の形式で出典を明示してください：\n"
			userPrompt += "- 検索されたドキュメントから: `> 📄 **システムドキュメントより:** [内容]`\n"
			userPrompt += "- 一般的な知識を補足する場合: `> 💡 **一般的な知識:** [内容]`\n\n"
		}

		// 過去の関連する会話履歴を追加
		if len(relevantHistory) > 0 {
			userPrompt += "\n## 📚 関連する過去の会話\n"
			for i, history := range relevantHistory {
				userPrompt += fmt.Sprintf("%d. %s\n", i+1, history)
			}
		}

		messages := []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		}

		// Azure OpenAI にリクエストを送信
		resp, err := aos.CreateChatCompletion(messages, 2000, 0.7)
		if err != nil {
			return "", fmt.Errorf("AI処理中にエラーが発生しました: %w", err)
		}

		if len(resp.Choices) > 0 {
			return resp.Choices[0].Message.Content, nil
		}

		return "", fmt.Errorf("AIから有効な回答が得られませんでした")
	}

	// 特殊コマンドのチェック（help, docsなど）
	if isSpecial, specialResponse := promptConfig.CheckSpecialCommand(chatMessage); isSpecial {
		return specialResponse, nil
	}

	// システムプロンプトを構築
	systemPrompt := promptConfig.BuildSystemPrompt()

	// ユーザープロンプトを構築
	userPrompt := fmt.Sprintf("## ユーザーからの質問\n%s\n", chatMessage)

	// 現在のコンテキストを最優先で追加（RAG検索結果）
	if context != "" {
		userPrompt += fmt.Sprintf("\n## 🔍 検索されたコンテキスト（必ずこれを基に回答してください）\n%s\n", context)
		userPrompt += "\n**重要**: 上記の検索結果に含まれる具体的な情報（コード、設定、フロー、数値など）を使用して回答してください。一般論ではなく、このシステム固有の情報を説明してください。\n"
		userPrompt += "\n**出典の明記**: 回答する際は、以下の形式で出典を明示してください：\n"
		userPrompt += "- 検索されたドキュメントから: `> 📄 **システムドキュメントより:** [内容]`\n"
		userPrompt += "- 一般的な知識を補足する場合: `> 💡 **一般的な知識:** [内容]`\n"
		userPrompt += "- 分析レポートから: `> 📊 **分析レポートより:** [内容]`\n"
		userPrompt += "- 過去の対話から: `> 🗣️ **過去の対話より:** [内容]`\n\n"
	}

	// 過去の関連する会話履歴を追加
	if len(relevantHistory) > 0 {
		userPrompt += "\n## 📚 関連する過去の会話\n"
		for i, history := range relevantHistory {
			userPrompt += fmt.Sprintf("%d. %s\n", i+1, history)
		}
	}

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// Azure OpenAI にリクエストを送信
	resp, err := aos.CreateChatCompletion(messages, 2000, 0.7)
	if err != nil {
		return "", fmt.Errorf("AI処理中にエラーが発生しました: %w", err)
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("AIから有効な回答が得られませんでした")
}

// ExtractMetadataFromMessage メッセージから意図やキーワードを抽出
func (aos *AzureOpenAIService) ExtractMetadataFromMessage(message string) (intent string, keywords []string, err error) {
	systemPrompt := `あなたはメッセージ分析の専門家です。与えられたメッセージから以下の情報を抽出してください：
1. 意図（intent）: "需要予測", "異常分析", "データ分析", "質問", "その他" のいずれか
2. キーワード: メッセージから重要なキーワードを3-5個抽出

レスポンスは以下のJSON形式で返してください：
{"intent": "意図", "keywords": ["キーワード1", "キーワード2", ...]}`

	userPrompt := fmt.Sprintf("以下のメッセージを分析してください：\n\n%s", message)

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	resp, err := aos.CreateChatCompletion(messages, 200, 0.3)
	if err != nil {
		return "", nil, fmt.Errorf("メタデータ抽出中にエラーが発生しました: %w", err)
	}

	if len(resp.Choices) > 0 {
		_ = resp.Choices[0].Message.Content
		// 簡易的なパース（本番環境では正規表現やJSON解析を使用）
		// ここでは基本的な実装として返す
		return "質問", []string{"需要予測", "分析"}, nil
	}

	return "", nil, fmt.Errorf("メタデータの抽出に失敗しました")
}

// ========================================
// 深掘り質問機能のための新しい関数
// ========================================

// EvaluateAnswerCompleteness ユーザーの回答を評価し、深掘りが必要か判定
func (aos *AzureOpenAIService) EvaluateAnswerCompleteness(
	anomalyContext string,
	question string,
	answer string,
	previousConversations []models.Conversation,
) (*models.AnswerEvaluation, error) {
	// 過去の会話履歴を構築
	conversationHistory := ""
	for i, conv := range previousConversations {
		conversationHistory += fmt.Sprintf("\n質問%d: %s", i+1, conv.Question)
		conversationHistory += fmt.Sprintf("\n回答%d: %s\n", i+1, conv.Answer)
	}

	systemPrompt := `あなたは需要予測システムの異常分析アシスタントです。
ユーザーが異常について回答した内容を評価し、さらに深掘り質問が必要かを判断してください。

【重要な注意事項】
- 異常データ（日付、製品ID、実績値、予測値、偏差）は既にシステムが把握しているため、これらを再度尋ねないこと
- 既に提供された情報を繰り返し尋ねないこと
- 深掘り質問は「WHY（なぜそうなったか）」「HOW（どのように対処したか）」「IMPACT（どの範囲に影響したか）」に焦点を当てる

【評価基準】
1. 具体性: 曖昧な表現ではなく、具体的な情報（固有名詞、期間、影響範囲など）が含まれているか
2. 因果関係: 原因と結果の関連性が明確に説明されているか
3. 実用性: 今後の需要予測や対策に活用できる情報か（パターン認識可能か）
4. 網羅性: 異常の全体像を把握するのに十分な情報か

【深掘りの判断】
- 完全性スコアが80点以上: 十分な情報が得られた（深掘り不要）
- 60-79点: もう1回深掘り質問をする価値あり
- 60点未満: 積極的に深掘り質問をすべき

【深掘り質問の例】
良い例：
- 「その天候不良は何日間続きましたか？」
- 「同じ天候パターンは過去にもありましたか？」
- 「競合の動きに気づいたきっかけは何ですか？」
- 「キャンペーンの対象製品や割引率を教えてください」

悪い例（既知の情報を尋ねている）：
- 「売上減少幅はどれくらいでしたか？」← 既にデータで分かっている
- 「何月何日でしたか？」← 既にデータで分かっている
- 「製品IDは何ですか？」← 既にデータで分かっている

【出力形式】
以下のJSON形式で必ず返してください：
{
  "is_sufficient": true/false,
  "completeness_score": 0-100の整数,
  "missing_aspects": ["欠けている情報1", "欠けている情報2"],
  "follow_up_question": "次に聞くべき質問（is_sufficient=falseの場合のみ。既知の情報は尋ねない）",
  "follow_up_choices": ["選択肢1", "選択肢2", "選択肢3", "選択肢4", "その他（自由記述）"],
  "reasoning": "判断理由の簡潔な説明",
  "suggested_tags": ["推奨タグ1", "推奨タグ2"],
  "suggested_impact": "positive/negative/neutral",
  "suggested_impact_value": 推定影響度（-100〜100の数値）
}`

	userPrompt := fmt.Sprintf(`【異常の状況】
%s

【これまでの会話】%s

【今回の質問】
%s

【ユーザーの回答】
%s

上記の回答を評価し、JSON形式で返してください。`,
		anomalyContext,
		conversationHistory,
		question,
		answer,
	)

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	resp, err := aos.CreateChatCompletion(messages, 1000, 0.7)
	if err != nil {
		return nil, fmt.Errorf("回答評価中にエラーが発生しました: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("AIから回答が得られませんでした")
	}

	content := resp.Choices[0].Message.Content

	// JSONを抽出（マークダウンのコードブロックに囲まれている場合に対応）
	jsonContent := content
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json") + 7
		end := strings.Index(content[start:], "```")
		if end > 0 {
			jsonContent = content[start : start+end]
		}
	} else if strings.Contains(content, "```") {
		start := strings.Index(content, "```") + 3
		end := strings.Index(content[start:], "```")
		if end > 0 {
			jsonContent = content[start : start+end]
		}
	}

	jsonContent = strings.TrimSpace(jsonContent)

	// JSONをパース
	var evaluation models.AnswerEvaluation
	if err := json.Unmarshal([]byte(jsonContent), &evaluation); err != nil {
		return nil, fmt.Errorf("AI回答のJSON解析に失敗しました: %w\nContent: %s", err, content)
	}

	return &evaluation, nil
}

// GenerateEnhancedQuestion は、コンテキストと仮説を活用した強化質問を生成
func (aos *AzureOpenAIService) GenerateEnhancedQuestion(
	anomaly models.AnomalyDetection,
	pastSimilarAnomalies []models.AnomalyResponse,
	userHistory []models.AnomalyResponse,
	weatherContext string,
	industryTrends string,
) (*models.EnhancedQuestion, error) {

	systemPrompt := `あなたは需要予測の専門家です。以下の情報を総合的に分析し、
ユーザーから最も価値のある情報を引き出す質問を生成してください。

【質問設計の原則】
1. 5W2H（What/Why/How/When/Where/Who/How Much）を網羅
2. 複数の仮説を立て、検証する質問を含める
3. 過去の類似ケースと比較し、パターンを特定する質問
4. 具体的な数値・固有名詞を引き出す
5. 予測モデルの改善に直結する情報を得る
6. データで既に分かっていることは尋ねない

【質問の構造】
- まず「データから見えていること」を3-5項目で箇条書き
- 過去の類似ケースがあれば参照
- 最低3つの仮説を立てる（信頼度付き）
- 各仮説を検証する具体的質問を用意
- 次の質問計画も示す

【出力形式】必ずこのJSON形式で返してください：
{
  "primary_question": "メインの質問文（状況説明+質問）",
  "context_summary": [
    "データポイント1",
    "データポイント2",
    "データポイント3"
  ],
  "past_patterns": "過去の類似ケースの説明（なければ「該当なし」）",
  "hypotheses": [
    {
      "id": "H1",
      "category": "internal",
      "title": "仮説のタイトル",
      "description": "詳細な説明",
      "confidence": 0.75,
      "data_evidence": "この仮説を支持するデータ",
      "verification_question": "検証質問",
      "choices": ["選択肢1", "選択肢2", "選択肢3", "選択肢4", "その他（自由記述）"],
      "expected_pattern": "期待される回答パターン"
    }
  ],
  "follow_up_plan": "次のステップで聞くべき内容"
}`

	// 過去の類似異常をフォーマット
	pastPatternsText := "該当なし"
	if len(pastSimilarAnomalies) > 0 {
		pastPatternsText = ""
		for i, past := range pastSimilarAnomalies[:min(3, len(pastSimilarAnomalies))] {
			pastPatternsText += fmt.Sprintf("\n%d. 日付: %s, 製品: %s, 原因: %s, 影響: %s",
				i+1, past.AnomalyDate, past.ProductID, past.Answer, past.Impact)
		}
	}

	// ユーザー履歴をフォーマット
	userHistoryText := "初回の質問です"
	if len(userHistory) > 0 {
		userHistoryText = "このユーザーの過去の回答傾向:\n"
		tagCounts := make(map[string]int)
		for _, hist := range userHistory {
			for _, tag := range hist.Tags {
				tagCounts[tag]++
			}
		}
		for tag, count := range tagCounts {
			userHistoryText += fmt.Sprintf("- %s: %d回\n", tag, count)
		}
	}

	userPrompt := fmt.Sprintf(`
【異常データ】
- 日付: %s
- 製品ID: %s
- 製品名: %s
- 実績値: %.2f
- 予測値: %.2f
- 偏差: %.1f%%
- 異常タイプ: %s
- 深刻度: %s

【過去の類似異常】%s

【このユーザーの回答傾向】
%s

【気象コンテキスト】
%s

【業界トレンド・その他の背景】
%s

上記を総合的に分析し、JSON形式で強化された質問を生成してください。`,
		anomaly.Date,
		anomaly.ProductID,
		anomaly.ProductName,
		anomaly.ActualValue,
		anomaly.ExpectedValue,
		anomaly.Deviation,
		anomaly.AnomalyType,
		anomaly.Severity,
		pastPatternsText,
		userHistoryText,
		weatherContext,
		industryTrends,
	)

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	resp, err := aos.CreateChatCompletion(messages, 2000, 0.7)
	if err != nil {
		return nil, fmt.Errorf("強化質問の生成に失敗しました: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("AIから回答が得られませんでした")
	}

	content := resp.Choices[0].Message.Content

	// JSONを抽出
	jsonContent := content
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json") + 7
		end := strings.Index(content[start:], "```")
		if end > 0 {
			jsonContent = content[start : start+end]
		}
	} else if strings.Contains(content, "```") {
		start := strings.Index(content, "```") + 3
		end := strings.Index(content[start:], "```")
		if end > 0 {
			jsonContent = content[start : start+end]
		}
	}

	jsonContent = strings.TrimSpace(jsonContent)

	// JSONをパース
	var enhancedQuestion models.EnhancedQuestion
	if err := json.Unmarshal([]byte(jsonContent), &enhancedQuestion); err != nil {
		return nil, fmt.Errorf("AIの回答JSON解析に失敗しました: %w\nContent: %s", err, content)
	}

	return &enhancedQuestion, nil
}

// GenerateScenarioQuestions は複数のシナリオ仮説を生成
func (aos *AzureOpenAIService) GenerateScenarioQuestions(
	anomaly models.AnomalyDetection,
	dataContext string,
) ([]models.ScenarioQuestion, error) {

	systemPrompt := `あなたは需要予測アナリストです。
異常データから複数の「もっともらしいシナリオ（仮説）」を生成し、
それぞれを検証する質問を作成してください。

【シナリオ生成の原則】
1. 最低3つ、最大5つの仮説を立てる
2. 各仮説に「信頼度スコア」を付与（データの裏付けの強さ: 0.0-1.0）
3. 仮説同士は排他的でなくてOK（複数が同時に成り立つこともある）
4. 各仮説を検証するための具体的質問を2-3個用意

【シナリオの分類】
- internal: 自社のアクション（キャンペーン、価格変更、在庫戦略、人員配置）
- external: 市場・競合（競合の動き、業界トレンド、規制変更、M&A）
- environmental: 環境要因（天候、災害、社会イベント、パンデミック）
- customer: 顧客行動（需要の本質的変化、トレンドシフト、ライフスタイル変化）

【出力形式】必ずこのJSON形式で返してください：
{
  "scenarios": [
    {
      "scenario_id": "S1",
      "category": "internal",
      "title": "シナリオのタイトル（30文字以内）",
      "description": "詳細な説明（100-200文字）",
      "confidence": 0.72,
      "data_evidence": "このシナリオを支持するデータの説明",
      "verification_questions": [
        {
          "question": "検証質問1",
          "choices": ["選択肢1", "選択肢2", "選択肢3", "その他"],
          "question_type": "single_choice",
          "expected_value": "選択肢1"
        }
      ],
      "related_scenarios": ["S2", "S3"]
    }
  ]
}`

	userPrompt := fmt.Sprintf(`
【異常データ】
- 日付: %s
- 製品ID: %s
- 製品名: %s
- 実績値: %.2f
- 予測値: %.2f
- 偏差: %.1f%%
- 異常タイプ: %s
- 深刻度: %s

【追加のデータコンテキスト】
%s

上記を分析し、複数のシナリオ仮説をJSON形式で生成してください。`,
		anomaly.Date,
		anomaly.ProductID,
		anomaly.ProductName,
		anomaly.ActualValue,
		anomaly.ExpectedValue,
		anomaly.Deviation,
		anomaly.AnomalyType,
		anomaly.Severity,
		dataContext,
	)

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	resp, err := aos.CreateChatCompletion(messages, 2000, 0.7)
	if err != nil {
		return nil, fmt.Errorf("シナリオ質問の生成に失敗しました: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("AIから回答が得られませんでした")
	}

	content := resp.Choices[0].Message.Content

	// JSONを抽出
	jsonContent := content
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json") + 7
		end := strings.Index(content[start:], "```")
		if end > 0 {
			jsonContent = content[start : start+end]
		}
	}
	jsonContent = strings.TrimSpace(jsonContent)

	// JSONをパース
	var result struct {
		Scenarios []models.ScenarioQuestion `json:"scenarios"`
	}
	if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
		return nil, fmt.Errorf("シナリオJSON解析に失敗しました: %w\nContent: %s", err, content)
	}

	return result.Scenarios, nil
}

// AnalyzeAnswerQuality は回答の品質を分析
func (aos *AzureOpenAIService) AnalyzeAnswerQuality(
	question string,
	answer string,
) (*models.AnswerAnalysis, error) {

	systemPrompt := `あなたは回答分析の専門家です。
ユーザーの回答を分析し、以下の観点で評価してください：

1. 具体性スコア（0-100）: 固有名詞、数値、具体的な事実がどれだけ含まれているか
2. 抽出されたエンティティ: 人名、組織名、場所、日付、数値、製品名など
3. 重要なフレーズ: キーワードとなるフレーズ（3-5個）
4. センチメント: positive（好影響）/neutral/negative（悪影響）
5. 実行可能性: この情報から具体的なアクションが取れるか
6. 予測価値: 予測モデルの改善にどれだけ貢献するか（0-100）

【出力形式】必ずこのJSON形式で返してください：
{
  "specificity_score": 75,
  "extracted_entities": [
    {"type": "organization", "value": "A社", "context": "競合他社"},
    {"type": "number", "value": "30", "context": "割引率30%"}
  ],
  "key_phrases": ["価格改定", "新商品投入", "在庫処分"],
  "sentiment": "positive",
  "actionable": true,
  "predictive_value": 80
}`

	userPrompt := fmt.Sprintf(`
【質問】
%s

【ユーザーの回答】
%s

上記の回答を分析し、JSON形式で結果を返してください。`,
		question,
		answer,
	)

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	resp, err := aos.CreateChatCompletion(messages, 1000, 0.5)
	if err != nil {
		return nil, fmt.Errorf("回答分析に失敗しました: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("AIから回答が得られませんでした")
	}

	content := resp.Choices[0].Message.Content
	jsonContent := content
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json") + 7
		end := strings.Index(content[start:], "```")
		if end > 0 {
			jsonContent = content[start : start+end]
		}
	}
	jsonContent = strings.TrimSpace(jsonContent)

	var analysis models.AnswerAnalysis
	if err := json.Unmarshal([]byte(jsonContent), &analysis); err != nil {
		return nil, fmt.Errorf("分析結果のJSON解析に失敗しました: %w\nContent: %s", err, content)
	}

	return &analysis, nil
}

// min は2つの整数の小さい方を返す
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
