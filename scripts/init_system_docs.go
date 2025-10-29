//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	config "hunt-chat-api/configs"
	"hunt-chat-api/pkg/services"

	"github.com/google/uuid"
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

	// コレクション名を固定
	const collectionName = "hunt_documents"

	// 先に既存のシステムドキュメントをすべて削除
	log.Printf("🗑️ コレクション '%s' から既存のシステムドキュメントを削除します...", collectionName)
	if err := vectorStoreService.DeleteDocumentsByType(ctx, collectionName, "system_documentation"); err != nil {
		// エラーが発生しても処理を続行する（コレクションが存在しない場合など）
		log.Printf("既存ドキュメントの削除中に警告: %v", err)
	} else {
		log.Println("✅ 既存のシステムドキュメントを削除しました。")
	}

	// システムドキュメントのリスト（新しいディレクトリ構造に対応）
	docs := []string{
		"README.md",
		"docs/api/API_MANUAL.md",
		"docs/guides/FILE_FORMAT_GUIDE.md",
		"docs/guides/AI_LEARNING_GUIDE.md",
		"docs/features/AI_QUESTION_STRATEGY.md",
		"docs/implementation/AI_QUESTION_IMPLEMENTATION.md",
		"docs/implementation/IMPLEMENTATION_SUMMARY.md",
		"docs/implementation/FINAL_IMPLEMENTATION_SUMMARY.md",
		"docs/implementation/ECONOMIC_CORRELATION_IMPLEMENTATION.md",
		"docs/features/CHAT_HISTORY_RAG.md",
		"docs/guides/WEEKLY_ANALYSIS_GUIDE.md",
		"docs/guides/DATA_AGGREGATION_GUIDE.md",
		"docs/guides/RAG_SYSTEM_GUIDE.md",
		"docs/guides/TROUBLESHOOTING_AND_BEST_PRACTICES.md",
		"docs/architecture/UML.md",
		"docs/project/要件定義.md",
		"docs/project/ワークフロー.md",
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

		docText := string(content)

		// 長い文書を分割 (約6000文字ごと、安全マージンを考慮)
		maxChunkSize := 6000
		chunks := splitDocument(docText, maxChunkSize)

		log.Printf("  📦 %d個のチャンクに分割", len(chunks))

		chunkSuccess := 0
		for i, chunk := range chunks {
			// ドキュメントIDをファイル名とチャンクインデックスから決定論的に生成
			idSource := fmt.Sprintf("%s-chunk-%d", docName, i)
			documentID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(idSource)).String()

			metadata := map[string]interface{}{
				"type":         "system_documentation",
				"file_name":    docName,
				"category":     getDocCategory(docName),
				"description":  getDocDescription(docName),
				"chunk_index":  i,
				"total_chunks": len(chunks),
			}

			err = vectorStoreService.StoreDocument(ctx, collectionName, documentID, chunk, metadata)
			if err != nil {
				log.Printf("  ⚠️ チャンク %d/%d の保存に失敗: %v", i+1, len(chunks), err)
				continue
			}
			chunkSuccess++
		}

		if chunkSuccess == len(chunks) {
			log.Printf("✅ 保存成功: %s (%d/%d チャンク)", docName, chunkSuccess, len(chunks))
			successCount++
		} else if chunkSuccess > 0 {
			log.Printf("⚠️ 部分的に成功: %s (%d/%d チャンク)", docName, chunkSuccess, len(chunks))
			successCount++
		} else {
			log.Printf("❌ ドキュメントの保存に失敗: %s - すべてのチャンクが失敗", docName)
			failCount++
		}
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
	// 新しいディレクトリ構造に基づいてカテゴリを判定
	if strings.Contains(filename, "docs/api/") {
		return "api"
	} else if strings.Contains(filename, "docs/architecture/") {
		return "architecture"
	} else if strings.Contains(filename, "docs/implementation/") {
		return "implementation"
	} else if strings.Contains(filename, "docs/guides/") {
		return "guides"
	} else if strings.Contains(filename, "docs/features/") {
		return "features"
	} else if strings.Contains(filename, "docs/project/") {
		return "project"
	}
	return "general"
}

// getDocDescription ドキュメントの説明を返す
func getDocDescription(filename string) string {
	descriptions := map[string]string{
		"README.md":                                                  "プロジェクトの概要とセットアップ手順",
		"docs/api/API_MANUAL.md":                                     "API利用マニュアル - エンドポイントと使用方法",
		"docs/guides/FILE_FORMAT_GUIDE.md":                           "ファイルアップロード形式ガイド - 必須列と形式の詳細",
		"docs/guides/AI_LEARNING_GUIDE.md":                           "AI学習システムのガイド - 回答保存と洞察取得",
		"docs/features/AI_QUESTION_STRATEGY.md":                      "AI質問力向上戦略ガイド - 多層質問、シナリオベース、継続質問の設計",
		"docs/implementation/AI_QUESTION_IMPLEMENTATION.md":          "AI質問機能の実装サンプル - 強化質問、仮説生成、回答品質分析のコード",
		"docs/implementation/IMPLEMENTATION_SUMMARY.md":              "実装概要 - システムアーキテクチャと技術スタック",
		"docs/implementation/FINAL_IMPLEMENTATION_SUMMARY.md":        "最終実装サマリー - 相関分析最適化とRAG活用",
		"docs/implementation/ECONOMIC_CORRELATION_IMPLEMENTATION.md": "経済データ相関分析実装 - 日経平均、為替、原油価格との相関",
		"docs/features/CHAT_HISTORY_RAG.md":                          "チャット履歴RAG機能の説明",
		"docs/guides/WEEKLY_ANALYSIS_GUIDE.md":                       "製品別分析機能の使い方（旧：週次分析）",
		"docs/guides/DATA_AGGREGATION_GUIDE.md":                      "データ集約分析ガイド - 日次・週次・月次分析",
		"docs/guides/RAG_SYSTEM_GUIDE.md":                            "RAG（検索拡張生成）システムガイド",
		"docs/guides/TROUBLESHOOTING_AND_BEST_PRACTICES.md":          "トラブルシューティングとベストプラクティス",
		"docs/architecture/UML.md":                                   "UML図とシステム設計・アーキテクチャ",
		"docs/project/要件定義.md":                                       "システムの要件定義書",
		"docs/project/ワークフロー.md":                                     "システムのワークフロー図",
	}

	if desc, ok := descriptions[filename]; ok {
		return desc
	}
	return "システムドキュメント"
}

// splitDocument 長い文書を指定サイズのチャンクに分割
func splitDocument(text string, maxSize int) []string {
	if len(text) <= maxSize {
		return []string{text}
	}

	var chunks []string
	lines := strings.Split(text, "\n")
	currentChunk := ""

	for _, line := range lines {
		// 次の行を追加すると制限を超える場合
		if len(currentChunk)+len(line)+1 > maxSize && currentChunk != "" {
			chunks = append(chunks, currentChunk)
			currentChunk = line + "\n"
		} else {
			if currentChunk != "" {
				currentChunk += "\n"
			}
			currentChunk += line
		}
	}

	// 最後のチャンクを追加
	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}
