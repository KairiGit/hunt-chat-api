package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	config "hunt-chat-api/configs"
	"hunt-chat-api/pkg/services"

	"github.com/joho/godotenv"
)

func main() {
	log.Println("🚀 システムドキュメントの初期化を開始します...")

	// .envファイルを読み込み
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// 設定を読み込む
	cfg := config.LoadConfig()

	// OpenAI サービスを初期化
	openaiService := services.NewAzureOpenAIService(
		cfg.AzureOpenAIEndpoint,
		cfg.AzureOpenAIAPIKey,
		cfg.AzureOpenAIAPIVersion,
		cfg.AzureOpenAIChatDeploymentName,
		cfg.AzureOpenAIEmbeddingDeploymentName,
	)

	// VectorStore サービスを初期化
	vectorStoreService, err := services.NewVectorStoreService(openaiService, cfg.QdrantURL, cfg.QdrantAPIKey)
	if err != nil {
		log.Fatalf("VectorStoreサービスの初期化に失敗: %v", err)
	}

	ctx := context.Background()

	// システムドキュメントのリスト
	docs := []string{
		"README.md",
		"API_MANUAL.md",
		"AI_LEARNING_GUIDE.md",
		"IMPLEMENTATION_SUMMARY.md",
		"CHAT_HISTORY_RAG.md",
		"WEEKLY_ANALYSIS_GUIDE.md",
		"TROUBLESHOOTING_AND_BEST_PRACTICES.md",
		"要件定義.md",
		"ワークフロー.md",
		"UML.md",
	}

	successCount := 0
	failCount := 0

	for _, docName := range docs {
		log.Printf("📄 処理中: %s", docName)

		// ファイルを読み込む
		content, err := os.ReadFile(docName)
		if err != nil {
			log.Printf("⚠️ ファイルの読み込みに失敗: %s - %v", docName, err)
			failCount++
			continue
		}

		// ドキュメントをベクトルDBに保存
		docID := fmt.Sprintf("system_doc_%s", docName)
		docText := string(content)
		metadata := map[string]interface{}{
			"type":        "system_documentation",
			"file_name":   docName,
			"category":    getDocCategory(docName),
			"description": getDocDescription(docName),
		}

		err = vectorStoreService.StoreDocument(ctx, docID, docText, docName, metadata)
		if err != nil {
			log.Printf("❌ ドキュメントの保存に失敗: %s - %v", docName, err)
			failCount++
			continue
		}

		log.Printf("✅ 保存成功: %s", docName)
		successCount++
	}

	separator := strings.Repeat("=", 50)
	log.Println("\n" + separator)
	log.Printf("📊 初期化完了")
	log.Printf("  成功: %d件", successCount)
	log.Printf("  失敗: %d件", failCount)
	log.Println(separator)

	if failCount > 0 {
		os.Exit(1)
	}
}

// getDocCategory ドキュメントのカテゴリを返す
func getDocCategory(filename string) string {
	switch filename {
	case "API_MANUAL.md":
		return "api"
	case "AI_LEARNING_GUIDE.md", "CHAT_HISTORY_RAG.md":
		return "ai"
	case "要件定義.md", "ワークフロー.md", "UML.md":
		return "design"
	case "IMPLEMENTATION_SUMMARY.md", "TROUBLESHOOTING_AND_BEST_PRACTICES.md":
		return "development"
	case "WEEKLY_ANALYSIS_GUIDE.md":
		return "usage"
	default:
		return "general"
	}
}

// getDocDescription ドキュメントの説明を返す
func getDocDescription(filename string) string {
	descriptions := map[string]string{
		"README.md":                             "プロジェクトの概要とセットアップ手順",
		"API_MANUAL.md":                         "API利用マニュアル - エンドポイントと使用方法",
		"AI_LEARNING_GUIDE.md":                  "AI学習システムのガイド - 回答保存と洞察取得",
		"IMPLEMENTATION_SUMMARY.md":             "実装概要 - システムアーキテクチャと技術スタック",
		"CHAT_HISTORY_RAG.md":                   "チャット履歴RAG機能の説明",
		"WEEKLY_ANALYSIS_GUIDE.md":              "週次分析機能の使い方",
		"TROUBLESHOOTING_AND_BEST_PRACTICES.md": "トラブルシューティングとベストプラクティス",
		"要件定義.md":                               "システムの要件定義書",
		"ワークフロー.md":                             "システムのワークフロー図",
		"UML.md":                                "UML図とシステム設計",
	}

	if desc, ok := descriptions[filename]; ok {
		return desc
	}
	return "システムドキュメント"
}
