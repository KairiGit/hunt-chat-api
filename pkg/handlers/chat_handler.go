package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"hunt-chat-api/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

// ChatInput RAGを使用したAIチャット
func (ah *AIHandler) ChatInput(c *gin.Context) {
	if ah.vectorStoreService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "データベースサービスが利用できません。設定を確認してください。",
		})
		return
	}
	var req ChatInputRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "リクエストの形式が正しくありません: " + err.Error()})
		return
	}
	if req.ChatMessage == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "チャットメッセージが必要です。"})
		return
	}

	// セッションIDが指定されていない場合は新規生成
	if req.SessionID == "" {
		req.SessionID = uuid.New().String()
	}

	ctx := c.Request.Context()

	// メタデータを抽出（意図やキーワード）
	intent, keywords, _ := ah.azureOpenAIService.ExtractMetadataFromMessage(req.ChatMessage)

	// ユーザーメッセージをチャット履歴として保存
	userEntry := models.ChatHistoryEntry{
		ID:        uuid.New().String(),
		SessionID: req.SessionID,
		UserID:    req.UserID,
		Role:      "user",
		Message:   req.ChatMessage,
		Context:   req.Context,
		Timestamp: time.Now().Format(time.RFC3339),
		Tags:      keywords,
		Metadata: models.Metadata{
			Intent:        intent,
			TopicKeywords: keywords,
		},
		CreatedAt: time.Now(),
	}

	// 非同期でチャット履歴を保存
	go func() {
		if err := ah.vectorStoreService.SaveChatHistory(context.Background(), userEntry); err != nil {
			log.Printf("ユーザーメッセージの履歴保存に失敗: %v", err)
		} else {
			log.Printf("✅ ユーザーメッセージを履歴に保存: SessionID=%s", req.SessionID)
		}
	}()

	// RAG: 類似した過去の会話を検索（チャット履歴から）
	var ragContext strings.Builder
	var relevantHistoryTexts []string
	var contextSources []models.ContextSource // スコア情報付きのソースリスト

	if req.Context != "" {
		ragContext.WriteString(req.Context) // ファイル分析のコンテキストを維持
		contextSources = append(contextSources, models.ContextSource{
			Type:     "file_analysis",
			FileName: "アップロードファイル",
			Score:    1.0, // 明示的に提供されたコンテキストは最高スコア
		})
	}

	// 🔍 過去のチャット履歴から関連する会話を検索
	chatHistory, err := ah.vectorStoreService.SearchChatHistory(ctx, req.ChatMessage, "", req.UserID, 3)
	if err != nil {
		log.Printf("チャット履歴検索に失敗: %v", err)
	} else if len(chatHistory) > 0 {
		ragContext.WriteString("\n\n## 過去の関連する会話履歴:\n")
		for i, entry := range chatHistory {
			historyText := fmt.Sprintf("[%s] %s: %s", entry.Timestamp, entry.Role, entry.Message)
			relevantHistoryTexts = append(relevantHistoryTexts, historyText)
			ragContext.WriteString(fmt.Sprintf("%d. %s (関連度: %.2f)\n", i+1, historyText, entry.Metadata.RelevanceScore))
			contextSources = append(contextSources, models.ContextSource{
				Type:     "chat_history",
				FileName: fmt.Sprintf("会話 %s", entry.Timestamp),
				Score:    float32(entry.Metadata.RelevanceScore),
				Date:     entry.Timestamp,
			})
		}
		log.Printf("📚 %d件の関連する過去の会話を取得しました", len(chatHistory))
	}

	// 🔍 統一コレクション 'hunt_documents' から関連ドキュメントを検索
	log.Println("🔍 統一コレクション 'hunt_documents' を検索します...")
	searchResults, err := ah.vectorStoreService.Search(ctx, req.ChatMessage, 3) // 検索件数を3に増やす
	if err != nil {
		log.Printf("ベクトル検索に失敗: %v", err)
	} else if len(searchResults) > 0 {
		ragContext.WriteString("\n\n## 関連ドキュメント情報:\n")
		log.Printf("📚 %d件の関連ドキュメントを 'hunt_documents' から取得しました", len(searchResults))
		for _, point := range searchResults {
			fileName := "不明なドキュメント"
			if fn, ok := point.Payload["file_name"]; ok {
				fileName = fn.GetStringValue()
			}

			// コンテキストソースにも詳細を追加
			contextSources = append(contextSources, models.ContextSource{
				Type:     "document",
				FileName: fileName,
				Score:    point.Score,
			})

			if textPayload, ok := point.Payload["text"]; ok {
				if text, ok := textPayload.GetKind().(*qdrant.Value_StringValue); ok {
					ragContext.WriteString(fmt.Sprintf("- [%s] %s (類似度: %.2f)\n", fileName, text.StringValue, point.Score))
				}
			}
		}
	}

	// 分析レポートを検索（質問が分析関連の場合）
	if strings.Contains(strings.ToLower(req.ChatMessage), "分析") ||
		strings.Contains(strings.ToLower(req.ChatMessage), "相関") ||
		strings.Contains(strings.ToLower(req.ChatMessage), "ファイル") ||
		strings.Contains(strings.ToLower(req.ChatMessage), "レポート") {

		analysisResults, err := ah.vectorStoreService.SearchAnalysisReports(ctx, req.ChatMessage, 2)
		if err != nil {
			log.Printf("分析レポート検索に失敗: %v", err)
		} else if len(analysisResults) > 0 {
			ragContext.WriteString("\n\n## 関連する過去の分析レポート:\n")
			for _, point := range analysisResults {
				if textPayload, ok := point.Payload["text"]; ok {
					if text, ok := textPayload.GetKind().(*qdrant.Value_StringValue); ok {
						var report models.AnalysisReport
						if json.Unmarshal([]byte(text.StringValue), &report) == nil {
							ragContext.WriteString(fmt.Sprintf("\n### レポート: %s\n", report.FileName))
							ragContext.WriteString(fmt.Sprintf("- 分析日: %s\n", report.AnalysisDate))
							ragContext.WriteString(fmt.Sprintf("- データ点数: %d\n", report.DataPoints))
							ragContext.WriteString(fmt.Sprintf("- サマリー:\n%s\n", report.Summary))
							if len(report.Correlations) > 0 {
								ragContext.WriteString("- 相関分析結果:\n")
								for _, corr := range report.Correlations {
									ragContext.WriteString(fmt.Sprintf("  * %s: %.3f (%s)\n",
										corr.Factor, corr.CorrelationCoef, corr.Interpretation))
								}
							}
							if report.Regression != nil {
								ragContext.WriteString(fmt.Sprintf("- 回帰分析: %s\n", report.Regression.Description))
							}
							contextSources = append(contextSources, models.ContextSource{
								Type:     "analysis_report",
								FileName: report.FileName,
								Score:    point.Score,
								Date:     report.AnalysisDate,
							})
						}
					}
				}
			}
		}
	}

	// 🤖 AIに応答を生成させる（過去の履歴を活用）
	aiResponse, err := ah.azureOpenAIService.ProcessChatWithHistory(
		req.ChatMessage,
		ragContext.String(),
		relevantHistoryTexts,
	)
	if err != nil {
		log.Printf("AI処理エラー詳細: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI処理中にエラーが発生しました: " + err.Error()})
		return
	}

	// AIの応答をチャット履歴として保存
	assistantEntry := models.ChatHistoryEntry{
		ID:        uuid.New().String(),
		SessionID: req.SessionID,
		UserID:    req.UserID,
		Role:      "assistant",
		Message:   aiResponse,
		Context:   req.Context,
		Timestamp: time.Now().Format(time.RFC3339),
		Tags:      keywords,
		Metadata: models.Metadata{
			Intent:        intent,
			TopicKeywords: keywords,
		},
		CreatedAt: time.Now(),
	}

	// 非同期でAI応答を履歴に保存
	go func() {
		if err := ah.vectorStoreService.SaveChatHistory(context.Background(), assistantEntry); err != nil {
			log.Printf("AI応答の履歴保存に失敗: %v", err)
		} else {
			log.Printf("✅ AI応答を履歴に保存: SessionID=%s", req.SessionID)
		}
	}()

	// レスポンスを返す（履歴情報を含む）
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"response": gin.H{
			"text":               aiResponse,
			"session_id":         req.SessionID,
			"relevant_history":   relevantHistoryTexts,
			"context_sources":    contextSources,
			"conversation_count": len(chatHistory),
		},
	})
}
