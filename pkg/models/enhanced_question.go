package models

// EnhancedQuestion 強化された質問構造
type EnhancedQuestion struct {
	PrimaryQuestion string          `json:"primary_question"` // メインの質問文
	ContextSummary  []string        `json:"context_summary"`  // データから見えていること
	PastPatterns    string          `json:"past_patterns"`    // 過去の類似ケース
	Hypotheses      []Hypothesis    `json:"hypotheses"`       // 複数の仮説
	FollowUpPlan    string          `json:"follow_up_plan"`   // 次の質問の計画
	QuestionLayers  []QuestionLayer `json:"question_layers"`  // 5W2H構造の質問
}

// Hypothesis 仮説構造
type Hypothesis struct {
	ID                   string   `json:"id"`                    // 仮説ID (H1, H2, ...)
	Category             string   `json:"category"`              // internal/external/environmental/customer
	Title                string   `json:"title"`                 // 仮説のタイトル
	Description          string   `json:"description"`           // 詳細な説明
	Confidence           float64  `json:"confidence"`            // 信頼度スコア (0.0-1.0)
	DataEvidence         string   `json:"data_evidence"`         // この仮説を支持するデータ
	VerificationQuestion string   `json:"verification_question"` // 検証質問
	Choices              []string `json:"choices"`               // 選択肢
	ExpectedPattern      string   `json:"expected_pattern"`      // 期待される回答パターン
}

// QuestionLayer 5W2H構造の質問レイヤー
type QuestionLayer struct {
	Layer           string   `json:"layer"`            // WHAT/WHY/HOW/WHEN/WHERE/WHO/HOW_MUCH/HOW_TO_PREVENT
	Question        string   `json:"question"`         // 質問文
	Choices         []string `json:"choices"`          // 選択肢
	Priority        int      `json:"priority"`         // 優先度 (1が最優先)
	RequiredContext []string `json:"required_context"` // 必要なコンテキスト
	Purpose         string   `json:"purpose"`          // この質問の目的
}

// ScenarioQuestion シナリオベースの質問
type ScenarioQuestion struct {
	ScenarioID            string                 `json:"scenario_id"`
	Category              string                 `json:"category"` // internal/external/environmental/customer
	Title                 string                 `json:"title"`
	Description           string                 `json:"description"`
	Confidence            float64                `json:"confidence"`
	DataEvidence          string                 `json:"data_evidence"`
	VerificationQuestions []VerificationQuestion `json:"verification_questions"`
	RelatedScenarios      []string               `json:"related_scenarios"` // 関連するシナリオID
}

// VerificationQuestion 検証質問
type VerificationQuestion struct {
	Question      string   `json:"question"`
	Choices       []string `json:"choices"`
	QuestionType  string   `json:"question_type"`  // single_choice/multiple_choice/free_text
	ExpectedValue string   `json:"expected_value"` // この回答が期待される場合、仮説の信頼度UP
}

// FollowUpQuestion フォローアップ質問
type FollowUpQuestion struct {
	ID              string   `json:"id"`
	AnomalyID       string   `json:"anomaly_id"`
	SessionID       string   `json:"session_id"`
	QuestionType    string   `json:"question_type"` // short_term_effect/medium_term_effect/long_term_pattern/yearly_review
	ScheduledDate   string   `json:"scheduled_date"`
	Status          string   `json:"status"` // pending/sent/answered/skipped
	Question        string   `json:"question"`
	Choices         []string `json:"choices"`
	Context         string   `json:"context"`          // 元の異常の情報
	PreviousAnswers []string `json:"previous_answers"` // 過去の回答
	Priority        int      `json:"priority"`         // 優先度
}

// QuestionQualityMetrics 質問の品質指標
type QuestionQualityMetrics struct {
	QuestionID            string  `json:"question_id"`
	ResponseRate          float64 `json:"response_rate"`          // 回答率
	SpecificityScore      float64 `json:"specificity_score"`      // 具体性スコア (0-100)
	InformationDensity    float64 `json:"information_density"`    // 情報密度（有用情報数/文字数）
	FollowUpSuccessRate   float64 `json:"follow_up_success_rate"` // 深掘り成功率
	PredictionImprovement float64 `json:"prediction_improvement"` // 予測精度改善度 (MAPE減少率)
	AverageAnswerLength   int     `json:"average_answer_length"`  // 平均回答文字数
	UserSatisfaction      float64 `json:"user_satisfaction"`      // ユーザー満足度 (1-5)
}

// AnswerAnalysis 回答の分析結果
type AnswerAnalysis struct {
	SpecificityScore  int      `json:"specificity_score"`  // 0-100
	ExtractedEntities []Entity `json:"extracted_entities"` // 抽出された固有名詞・数値
	KeyPhrases        []string `json:"key_phrases"`        // 重要なフレーズ
	Sentiment         string   `json:"sentiment"`          // positive/neutral/negative
	Actionable        bool     `json:"actionable"`         // 実行可能な情報か
	PredictiveValue   int      `json:"predictive_value"`   // 予測への貢献度 (0-100)
}

// Entity 抽出されたエンティティ
type Entity struct {
	Type    string `json:"type"`    // person/organization/location/date/number/product
	Value   string `json:"value"`   // 実際の値
	Context string `json:"context"` // 文脈
}
