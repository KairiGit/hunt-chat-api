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

	"github.com/joho/godotenv"
)

func main() {
	log.Println("🧹 システムドキュメントコレクションのクリーンアップを開始します...")

	// .env.localファイルを優先的に読み込み（本番環境用）
	if err := godotenv.Load(".env.local"); err != nil {
		log.Printf("Warning: .env.local file not found, trying .env: %v", err)
		if err := godotenv.Load(); err != nil {
			log.Printf("Warning: .env file not found: %v", err)
		}
	}

	// 設定を読み込む
	cfg := config.LoadConfig()

	log.Printf("接続先Qdrant: %s", cfg.QdrantURL)

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

	// すべてのコレクションを取得
	collections, err := vectorStoreService.ListCollections(ctx)
	if err != nil {
		log.Fatalf("コレクション一覧の取得に失敗: %v", err)
	}

	log.Printf("📋 全コレクション数: %d", len(collections))

	// system_doc_ で始まるコレクションをフィルタリング
	var systemDocCollections []string
	for _, collection := range collections {
		if strings.HasPrefix(collection, "system_doc_") {
			systemDocCollections = append(systemDocCollections, collection)
		}
	}

	if len(systemDocCollections) == 0 {
		log.Println("✅ system_doc_ で始まるコレクションは見つかりませんでした")
		return
	}

	log.Printf("🔍 削除対象のコレクション: %d件", len(systemDocCollections))
	for _, collection := range systemDocCollections {
		log.Printf("  - %s", collection)
	}

	// 確認プロンプト
	fmt.Print("\n❓ これらのコレクションを削除してもよろしいですか？ (yes/no): ")
	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "yes" {
		log.Println("❌ 削除をキャンセルしました")
		os.Exit(0)
	}

	// 削除実行
	deletedCount := 0
	failedCount := 0

	for _, collection := range systemDocCollections {
		log.Printf("🗑️  削除中: %s", collection)
		if err := vectorStoreService.DeleteCollection(ctx, collection); err != nil {
			log.Printf("  ⚠️ 削除失敗: %v", err)
			failedCount++
		} else {
			log.Printf("  ✅ 削除成功")
			deletedCount++
		}
	}

	log.Println("\n==================================================")
	log.Printf("📊 クリーンアップ完了")
	log.Printf("  削除成功: %d件", deletedCount)
	log.Printf("  削除失敗: %d件", failedCount)
	log.Println("==================================================")

	if failedCount > 0 {
		os.Exit(1)
	}
}
