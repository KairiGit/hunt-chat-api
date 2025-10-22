# AI質問力向上 - 実装サンプルコード

## 📝 使用例

### 1. 強化された質問生成の使用例

```go
// pkg/handlers/ai_handler.go に追加

// GenerateEnhancedAnomalyQuestion は強化された異常質問を生成
func (ah *AIHandler) GenerateEnhancedAnomalyQuestion(c *gin.Context) {
	// 異常IDを取得
	anomalyID := c.Query("anomaly_id")
	if anomalyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "anomaly_idが指定されていません",
		})
		return
	}

	// 異常データを取得（実際の実装では、Qdrantやメモリから取得）
	// ここではサンプルデータ
	anomaly := models.AnomalyDetection{
		Date:          "2024-03-15",
		ProductID:     "P001",
		ProductName:   "製品A",
		ActualValue:   1500,
		ExpectedValue: 1000,
		Deviation:     50.0,
		AnomalyType:   "急増",
		Severity:      "high",
	}

	ctx := c.Request.Context()

	// 過去の類似異常を取得
	pastAnomalies, err := ah.vectorStoreService.SearchSimilarAnomalies(
		ctx,
		anomaly.ProductID,
		anomaly.Date,
		3, // 上位3件
	)
	if err != nil {
		log.Printf("過去の異常取得エラー: %v", err)
		pastAnomalies = []models.AnomalyResponse{} // 空でも続行
	}

	// ユーザー履歴を取得
	userID := c.GetString("user_id")
	userHistory, err := ah.vectorStoreService.GetUserAnomalyHistory(ctx, userID, 10)
	if err != nil {
		log.Printf("ユーザー履歴取得エラー: %v", err)
		userHistory = []models.AnomalyResponse{}
	}

	// 気象コンテキストを取得
	weatherContext := ""
	if ah.weatherService != nil {
		weather, err := ah.weatherService.GetWeatherByDate(anomaly.Date)
		if err == nil {
			weatherContext = fmt.Sprintf(
				"気温: %.1f°C, 天候: %s, 降水量: %.1fmm",
				weather.Temperature,
				weather.Condition,
				weather.Precipitation,
			)
		}
	}

	// 業界トレンド（実際にはAPIから取得、ここでは簡易版）
	industryTrends := "同業界全体で3月は前年比+15%の成長傾向"

	// 強化された質問を生成
	enhancedQuestion, err := ah.azureOpenAIService.GenerateEnhancedQuestion(
		anomaly,
		pastAnomalies,
		userHistory,
		weatherContext,
		industryTrends,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "質問生成に失敗しました: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"question": enhancedQuestion,
		"message":  "強化された質問を生成しました",
	})
}
```

### 2. シナリオベース質問の生成

```go
// GenerateScenarioBasedQuestion はシナリオ仮説を生成
func (ah *AIHandler) GenerateScenarioBasedQuestion(c *gin.Context) {
	anomalyID := c.Query("anomaly_id")
	
	// 異常データを取得（省略）
	anomaly := getAnomalyByID(anomalyID)

	// データコンテキストを構築
	dataContext := fmt.Sprintf(`
【同カテゴリの他製品】
- 製品B: 通常通り（変化なし）
- 製品C: 通常通り（変化なし）
→ この製品だけが特異的に変動

【新規顧客比率】
- 通常: 30%%
- 今回: 55%%
→ 既存顧客以外の購入が急増

【時間帯別売上】
- 午前: +10%%
- 午後: +80%%
→ 特定時間帯に集中

【購入パターン】
- 単品購入: 通常通り
- まとめ買い: +120%%
→ 購買行動に変化
	`)

	// シナリオ質問を生成
	scenarios, err := ah.azureOpenAIService.GenerateScenarioQuestions(
		anomaly,
		dataContext,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "シナリオ質問生成に失敗しました: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"scenarios": scenarios,
		"message":   fmt.Sprintf("%d個のシナリオ仮説を生成しました", len(scenarios)),
	})
}
```

### 3. 回答品質の分析

```go
// AnalyzeUserAnswer はユーザーの回答品質を分析
func (ah *AIHandler) AnalyzeUserAnswer(c *gin.Context) {
	var req struct {
		Question string `json:"question" binding:"required"`
		Answer   string `json:"answer" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "リクエストが不正です: " + err.Error(),
		})
		return
	}

	// 回答を分析
	analysis, err := ah.azureOpenAIService.AnalyzeAnswerQuality(
		req.Question,
		req.Answer,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "回答分析に失敗しました: " + err.Error(),
		})
		return
	}

	// 具体性スコアが低い場合は深掘り推奨
	needsFollowUp := analysis.SpecificityScore < 60

	response := gin.H{
		"success":         true,
		"analysis":        analysis,
		"needs_follow_up": needsFollowUp,
	}

	if needsFollowUp {
		response["message"] = "回答がやや曖昧です。もう少し具体的に教えていただけますか？"
		response["suggestions"] = []string{
			"具体的な数値（金額、数量、期間など）",
			"固有名詞（製品名、企業名、人名など）",
			"時系列の詳細（いつから、いつまで、何日間など）",
		}
	} else {
		response["message"] = "詳細な情報をありがとうございます！"
	}

	c.JSON(http.StatusOK, response)
}
```

### 4. フォローアップ質問スケジューラー

```go
// pkg/services/follow_up_scheduler.go

package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"hunt-chat-api/pkg/models"
)

type FollowUpScheduler struct {
	db                 *sql.DB
	azureOpenAIService *AzureOpenAIService
}

func NewFollowUpScheduler(db *sql.DB, aiService *AzureOpenAIService) *FollowUpScheduler {
	return &FollowUpScheduler{
		db:                 db,
		azureOpenAIService: aiService,
	}
}

// ScheduleFollowUps は異常回答後のフォローアップを計画
func (fus *FollowUpScheduler) ScheduleFollowUps(
	anomalyID string,
	sessionID string,
	initialResponse models.AnomalyResponse,
) error {
	schedules := []struct {
		delayDays    int
		questionType string
		description  string
	}{
		{7, "short_term_effect", "1週間後の状況確認"},
		{30, "medium_term_effect", "1ヶ月後の効果測定"},
		{90, "long_term_pattern", "3ヶ月後のパターン確認"},
		{365, "yearly_review", "1年後の年次レビュー"},
	}

	for _, schedule := range schedules {
		scheduledDate := time.Now().AddDate(0, 0, schedule.delayDays)

		followUp := models.FollowUpQuestion{
			AnomalyID:       anomalyID,
			SessionID:       sessionID,
			QuestionType:    schedule.questionType,
			ScheduledDate:   scheduledDate.Format("2006-01-02"),
			Status:          "pending",
			Context:         formatInitialResponse(initialResponse),
			PreviousAnswers: []string{initialResponse.Answer},
			Priority:        calculatePriority(initialResponse, schedule.questionType),
		}

		// データベースに保存（テーブル作成は別途必要）
		err := fus.saveFollowUp(followUp)
		if err != nil {
			return fmt.Errorf("フォローアップ保存エラー: %w", err)
		}
	}

	return nil
}

// GetPendingFollowUps は今日実行すべきフォローアップを取得
func (fus *FollowUpScheduler) GetPendingFollowUps() ([]models.FollowUpQuestion, error) {
	today := time.Now().Format("2006-01-02")

	query := `
		SELECT id, anomaly_id, session_id, question_type, scheduled_date, 
		       context, previous_answers, priority
		FROM follow_up_questions
		WHERE scheduled_date <= ? AND status = 'pending'
		ORDER BY priority DESC, scheduled_date ASC
		LIMIT 50
	`

	rows, err := fus.db.Query(query, today)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var followUps []models.FollowUpQuestion
	for rows.Next() {
		var fu models.FollowUpQuestion
		// スキャン処理（省略）
		followUps = append(followUps, fu)
	}

	return followUps, nil
}

// GenerateFollowUpQuestion はフォローアップ質問を生成
func (fus *FollowUpScheduler) GenerateFollowUpQuestion(
	followUp models.FollowUpQuestion,
) (string, []string, error) {
	var questionPrompt string

	switch followUp.QuestionType {
	case "short_term_effect":
		questionPrompt = fmt.Sprintf(`
先週、以下の異常について回答いただきました：
%s

【前回の回答】
%s

1週間経過しましたが、その後の状況を教えてください：
- 異常は継続していますか？それとも収束しましたか？
- 実施した対策の効果はありましたか？
- 新たな気づきや変化はありますか？
		`, followUp.Context, followUp.PreviousAnswers[0])

	case "medium_term_effect":
		questionPrompt = fmt.Sprintf(`
1ヶ月前に以下の異常について回答いただきました：
%s

【前回の回答】
%s

1ヶ月経過しましたが、中期的な効果を教えてください：
- 売上・利益への影響はどうでしたか？
- 対策は定着していますか？
- 予想外の副作用や新たな課題はありましたか？
		`, followUp.Context, followUp.PreviousAnswers[0])

	case "long_term_pattern":
		questionPrompt = fmt.Sprintf(`
3ヶ月前に以下の異常について回答いただきました：
%s

【前回の回答】
%s

3ヶ月の長期的視点で振り返ってください：
- 同様のパターンは再発しましたか？
- 季節性や周期性は見られましたか？
- 学んだことを今後にどう活かしますか？
		`, followUp.Context, followUp.PreviousAnswers[0])

	case "yearly_review":
		questionPrompt = fmt.Sprintf(`
1年前に以下の異常について回答いただきました：
%s

【前回の回答】
%s

1年間の総括をお願いします：
- この異常から得た最大の学びは何ですか？
- 同じ状況が再度起きた場合、どう対応しますか？
- 予測精度を上げるために必要な情報は何ですか？
		`, followUp.Context, followUp.PreviousAnswers[0])
	}

	choices := []string{
		"継続中",
		"収束した",
		"別の問題に発展した",
		"特に変化なし",
		"その他（自由記述）",
	}

	return questionPrompt, choices, nil
}

// ヘルパー関数
func formatInitialResponse(resp models.AnomalyResponse) string {
	return fmt.Sprintf(
		"日付: %s, 製品: %s, 原因: %s",
		resp.AnomalyDate,
		resp.ProductID,
		resp.Answer,
	)
}

func calculatePriority(resp models.AnomalyResponse, questionType string) int {
	// 影響度が高いほど優先度UP
	priority := 50
	if resp.ImpactValue > 50 {
		priority += 30
	} else if resp.ImpactValue > 20 {
		priority += 15
	}

	// 質問タイプによる調整
	switch questionType {
	case "short_term_effect":
		priority += 20 // 短期は最優先
	case "medium_term_effect":
		priority += 10
	}

	return priority
}

func (fus *FollowUpScheduler) saveFollowUp(followUp models.FollowUpQuestion) error {
	// データベース保存処理（省略）
	return nil
}
```

### 5. ルーティング設定

```go
// cmd/server/main.go または routes設定ファイルに追加

// 強化された質問生成
router.GET("/api/v1/ai/enhanced-question", aiHandler.GenerateEnhancedAnomalyQuestion)

// シナリオベース質問
router.GET("/api/v1/ai/scenario-questions", aiHandler.GenerateScenarioBasedQuestion)

// 回答品質分析
router.POST("/api/v1/ai/analyze-answer", aiHandler.AnalyzeUserAnswer)

// フォローアップ質問
router.GET("/api/v1/ai/follow-up-questions", func(c *gin.Context) {
	scheduler := services.NewFollowUpScheduler(db, aiHandler.azureOpenAIService)
	followUps, err := scheduler.GetPendingFollowUps()
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "フォローアップ取得に失敗しました",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"follow_ups": followUps,
		"count":      len(followUps),
	})
})
```

### 6. データベーススキーマ（follow_up_questions テーブル）

```sql
CREATE TABLE IF NOT EXISTS follow_up_questions (
    id VARCHAR(255) PRIMARY KEY,
    anomaly_id VARCHAR(255) NOT NULL,
    session_id VARCHAR(255) NOT NULL,
    question_type VARCHAR(50) NOT NULL,
    scheduled_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    question TEXT,
    choices JSON,
    context TEXT,
    previous_answers JSON,
    priority INT DEFAULT 50,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    sent_at TIMESTAMP NULL,
    answered_at TIMESTAMP NULL,
    answer TEXT NULL,
    
    INDEX idx_scheduled_date (scheduled_date),
    INDEX idx_status (status),
    INDEX idx_priority (priority)
);

-- フォローアップ質問の実行履歴
CREATE TABLE IF NOT EXISTS follow_up_executions (
    id VARCHAR(255) PRIMARY KEY,
    follow_up_id VARCHAR(255) NOT NULL,
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    answer TEXT,
    answer_quality_score INT,
    predictive_value INT,
    
    FOREIGN KEY (follow_up_id) REFERENCES follow_up_questions(id)
);
```

## 📊 効果測定の実装

### 質問品質メトリクスの計算

```go
// pkg/services/question_metrics.go

package services

import (
	"hunt-chat-api/pkg/models"
	"strings"
)

type QuestionMetricsService struct {
	azureOpenAIService *AzureOpenAIService
}

func NewQuestionMetricsService(aiService *AzureOpenAIService) *QuestionMetricsService {
	return &QuestionMetricsService{
		azureOpenAIService: aiService,
	}
}

// CalculateQuestionMetrics は質問の効果を測定
func (qms *QuestionMetricsService) CalculateQuestionMetrics(
	questionID string,
	responses []models.AnomalyResponse,
	baselineMAPE float64,
	improvedMAPE float64,
) models.QuestionQualityMetrics {

	totalAsked := len(responses)
	totalAnswered := 0
	totalSpecificity := 0.0
	totalLength := 0

	for _, resp := range responses {
		if resp.Answer != "" {
			totalAnswered++
			
			// 簡易的な具体性スコア計算
			specificity := calculateSpecificity(resp.Answer)
			totalSpecificity += specificity
			totalLength += len(resp.Answer)
		}
	}

	responseRate := 0.0
	if totalAsked > 0 {
		responseRate = float64(totalAnswered) / float64(totalAsked) * 100
	}

	avgSpecificity := 0.0
	if totalAnswered > 0 {
		avgSpecificity = totalSpecificity / float64(totalAnswered)
	}

	avgLength := 0
	if totalAnswered > 0 {
		avgLength = totalLength / totalAnswered
	}

	// 予測精度改善度
	predictionImprovement := 0.0
	if baselineMAPE > 0 {
		predictionImprovement = ((baselineMAPE - improvedMAPE) / baselineMAPE) * 100
	}

	return models.QuestionQualityMetrics{
		QuestionID:            questionID,
		ResponseRate:          responseRate,
		SpecificityScore:      avgSpecificity,
		InformationDensity:    calculateInformationDensity(responses),
		FollowUpSuccessRate:   calculateFollowUpSuccess(responses),
		PredictionImprovement: predictionImprovement,
		AverageAnswerLength:   avgLength,
		UserSatisfaction:      4.2, // フィードバックから取得（仮）
	}
}

func calculateSpecificity(answer string) float64 {
	score := 0.0

	// 固有名詞の検出（簡易版）
	if strings.Contains(answer, "株式会社") || strings.Contains(answer, "社") {
		score += 20
	}

	// 数値の検出
	hasNumber := false
	for _, char := range answer {
		if char >= '0' && char <= '9' {
			hasNumber = true
			break
		}
	}
	if hasNumber {
		score += 20
	}

	// 具体的な日付表現
	if strings.Contains(answer, "月") && strings.Contains(answer, "日") {
		score += 15
	}

	// 長さボーナス（詳細な回答ほど高評価）
	if len(answer) > 100 {
		score += 15
	} else if len(answer) > 50 {
		score += 10
	}

	// 曖昧表現のペナルティ
	vagueTerms := []string{"たぶん", "おそらく", "かもしれない", "よく分からない"}
	for _, term := range vagueTerms {
		if strings.Contains(answer, term) {
			score -= 10
		}
	}

	// 0-100にクリップ
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

func calculateInformationDensity(responses []models.AnomalyResponse) float64 {
	// 有用情報数 / 総文字数
	totalInfo := 0
	totalChars := 0

	for _, resp := range responses {
		if resp.Answer == "" {
			continue
		}

		totalChars += len(resp.Answer)

		// タグ数（カテゴリ分類された情報数）
		totalInfo += len(resp.Tags)

		// 数値の出現回数
		for _, char := range resp.Answer {
			if char >= '0' && char <= '9' {
				totalInfo++
				break // 1回答につき1カウント
			}
		}
	}

	if totalChars == 0 {
		return 0
	}

	return float64(totalInfo) / float64(totalChars) * 1000 // 1000文字あたりの情報数
}

func calculateFollowUpSuccess(responses []models.AnomalyResponse) float64 {
	// 深掘り質問で新情報が得られた割合
	// 実装簡略化のため固定値を返す（実際はセッションデータから計算）
	return 65.0 // 65%の深掘りで新情報取得
}
```

## 🎓 使用イメージ（フロントエンド）

### React コンポーネント例

```typescript
// src/components/EnhancedQuestionCard.tsx

import React, { useState } from 'react';

interface Hypothesis {
  id: string;
  title: string;
  description: string;
  confidence: number;
  verification_question: string;
  choices: string[];
}

interface EnhancedQuestionProps {
  anomalyId: string;
}

const EnhancedQuestionCard: React.FC<EnhancedQuestionProps> = ({ anomalyId }) => {
  const [question, setQuestion] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [selectedHypothesis, setSelectedHypothesis] = useState<string | null>(null);
  const [answer, setAnswer] = useState<string>('');

  const fetchEnhancedQuestion = async () => {
    setLoading(true);
    try {
      const response = await fetch(
        `/api/v1/ai/enhanced-question?anomaly_id=${anomalyId}`
      );
      const data = await response.json();
      if (data.success) {
        setQuestion(data.question);
      }
    } catch (error) {
      console.error('質問取得エラー:', error);
    }
    setLoading(false);
  };

  React.useEffect(() => {
    fetchEnhancedQuestion();
  }, [anomalyId]);

  if (loading) {
    return <div>質問を生成中...</div>;
  }

  if (!question) {
    return null;
  }

  return (
    <div className="enhanced-question-card">
      <h3>📊 異常に関する質問</h3>

      {/* データから見えていること */}
      <div className="context-summary">
        <h4>💡 データから見えていること</h4>
        <ul>
          {question.context_summary?.map((item: string, index: number) => (
            <li key={index}>{item}</li>
          ))}
        </ul>
      </div>

      {/* 過去のパターン */}
      {question.past_patterns && question.past_patterns !== '該当なし' && (
        <div className="past-patterns">
          <h4>📜 過去の類似ケース</h4>
          <p>{question.past_patterns}</p>
        </div>
      )}

      {/* メイン質問 */}
      <div className="primary-question">
        <h4>❓ 質問</h4>
        <p>{question.primary_question}</p>
      </div>

      {/* 仮説一覧 */}
      <div className="hypotheses">
        <h4>🔍 考えられるシナリオ</h4>
        {question.hypotheses?.map((hyp: Hypothesis) => (
          <div key={hyp.id} className="hypothesis-card">
            <div className="hypothesis-header">
              <h5>{hyp.title}</h5>
              <span className="confidence">信頼度: {(hyp.confidence * 100).toFixed(0)}%</span>
            </div>
            <p>{hyp.description}</p>
            
            <div className="verification">
              <p><strong>{hyp.verification_question}</strong></p>
              <div className="choices">
                {hyp.choices.map((choice, idx) => (
                  <button
                    key={idx}
                    className={selectedHypothesis === `${hyp.id}-${idx}` ? 'selected' : ''}
                    onClick={() => {
                      setSelectedHypothesis(`${hyp.id}-${idx}`);
                      setAnswer(choice);
                    }}
                  >
                    {choice}
                  </button>
                ))}
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* 自由記述 */}
      <div className="free-text">
        <textarea
          placeholder="その他、詳しい状況があれば教えてください..."
          value={answer}
          onChange={(e) => setAnswer(e.target.value)}
          rows={4}
        />
      </div>

      {/* 送信ボタン */}
      <button
        className="submit-button"
        disabled={!answer}
        onClick={async () => {
          // 回答送信処理
          await fetch('/api/v1/ai/save-anomaly-response', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              anomaly_id: anomalyId,
              answer: answer,
              selected_hypothesis: selectedHypothesis,
            }),
          });
        }}
      >
        回答を送信
      </button>

      {/* 次のステップ */}
      {question.follow_up_plan && (
        <div className="follow-up-plan">
          <h4>📅 次のステップ</h4>
          <p>{question.follow_up_plan}</p>
        </div>
      )}
    </div>
  );
};

export default EnhancedQuestionCard;
```

---

## まとめ

この実装により、以下が実現されます：

1. ✅ **多層質問フレームワーク**: 5W2Hで段階的に深掘り
2. ✅ **コンテキスト活用**: 過去データ・ユーザー履歴を参照
3. ✅ **仮説駆動質問**: 複数のシナリオを提示して検証
4. ✅ **継続質問**: 1週間後、1ヶ月後、3ヶ月後、1年後の追跡
5. ✅ **品質測定**: 回答の具体性を評価し、深掘りの要否を判定
6. ✅ **予測精度向上**: 質問回答を学習データに活用

**次のステップ**: 
- ルーティングを設定してAPIを公開
- フロントエンドに統合
- A/Bテストで効果測定
