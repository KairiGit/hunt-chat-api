package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// .envファイルを読み込み
	if err := godotenv.Load(); err != nil {
		log.Fatalf("FATAL: .env file not found or could not be loaded: %v", err)
	}

	// 環境変数から設定を読み込み
	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	proxyURL := os.Getenv("AZURE_OPENAI_PROXY_URL")
	apiKey := os.Getenv("AZURE_OPENAI_API_KEY")
	chatDeploymentName := os.Getenv("AZURE_OPENAI_CHAT_DEPLOYMENT_NAME")
	apiVersion := os.Getenv("AZURE_OPENAI_API_VERSION")

	if endpoint == "" || apiKey == "" || chatDeploymentName == "" {
		log.Fatal("FATAL: 必要な環境変数 (AZURE_OPENAI_ENDPOINT, AZURE_OPENAI_API_KEY, AZURE_OPENAI_CHAT_DEPLOYMENT_NAME) が設定されていません。")
	}

	// --- HTTPクライアントのセットアップ ---
	transport := &http.Transport{}
	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxy)
			log.Println("INFO: HTTPクライアントにプロキシを設定しました:", proxyURL)
		} else {
			log.Printf("WARN: 無効なプロキシURLです。プロキシは使用されません: %v", err)
		}
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second, // タイムアウトを延長
	}

	// --- リクエストの作成 ---
	// 1. リクエストURL
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		strings.TrimSuffix(endpoint, "/"), chatDeploymentName, apiVersion)

	// 2. リクエストボディ
	requestBodyMap := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": "Hello!"},
		},
	}
	requestBody, _ := json.Marshal(requestBodyMap)

	log.Println("INFO: リクエストURL:", url)
	log.Println("INFO: リクエストボディ:", string(requestBody))

	// 3. HTTPリクエストオブジェクトの作成
	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Fatalf("FATAL: HTTPリクエストの作成に失敗: %v", err)
	}

	// 4. ヘッダーの設定
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", apiKey)

	// --- リクエストの実行 ---
	log.Println("INFO: リクエストを送信します...")
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatalf("FATAL: HTTPリクエストの実行に失敗: %v", err)
	}
	defer resp.Body.Close()

	// --- レスポンスの表示 ---
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("FATAL: レスポンスの読み取りに失敗: %v", err)
	}

	log.Println("--- レスポンス ---")
	log.Println("ステータスコード:", resp.StatusCode)
	log.Println("レスポンスボディ:", string(body))

	if resp.StatusCode == http.StatusOK {
		log.Println("\nSUCCESS: 正常に応答が返ってきました。")
	} else {
		log.Println("\nERROR: サーバーからエラーステータスが返されました。")
	}
}
