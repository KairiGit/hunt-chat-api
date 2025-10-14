package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"hunt-chat-api/pkg/models"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// VectorStoreService はQdrantとのやり取りを管理します
type VectorStoreService struct {
	qdrantClient            qdrant.PointsClient
	qdrantCollectionsClient qdrant.CollectionsClient
	azureOpenAIService      *AzureOpenAIService
}

// NewVectorStoreService は新しいVectorStoreServiceを初期化して返します
func NewVectorStoreService(azureOpenAIService *AzureOpenAIService, qdrantURL string, qdrantAPIKey string) *VectorStoreService {
	// 接続オプション
	var dialOpts []grpc.DialOption

	// APIキーの有無で、Cloud接続(TLS+APIキー)とローカル接続(非セキュア)を切り替える
	if qdrantAPIKey != "" {
		// --- Qdrant Cloud用の接続 --- //
		log.Println("Qdrant Cloud (TLS) への接続を準備します...")
		creds := credentials.NewTLS(&tls.Config{})
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))

		// APIキー認証インターセプタを追加
		authInterceptor := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			ctx = metadata.AppendToOutgoingContext(ctx, "api-key", qdrantAPIKey)
			return invoker(ctx, method, req, reply, cc, opts...)
		}
		dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(authInterceptor))

	} else {
		// --- ローカル用の接続 (以前成功した方式) --- //
		log.Println("ローカルのQdrant (非TLS) への接続を準備します...")
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// gRPC接続を確立
	conn, err := grpc.NewClient(qdrantURL, dialOpts...)
	if err != nil {
		log.Fatalf("QdrantへのgRPCクライアント作成に失敗しました: %v", err)
	}

	qdrantPointsClient := qdrant.NewPointsClient(conn)
	qdrantCollectionsClient := qdrant.NewCollectionsClient(conn)

	collectionName := "hunt_chat_documents"
	vectorSize := uint64(1536) // text-embedding-3-smallの次元数

	// Qdrantサーバーが完全に起動するまでリトライしながらコレクションの存在確認を行う
	maxRetries := 10
	retryInterval := 2 * time.Second
	var collectionExists bool
	var listErr error

	log.Println("Qdrantサーバーの準備を確認中...")
	for i := 0; i < maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		res, err := qdrantCollectionsClient.List(ctx, &qdrant.ListCollectionsRequest{})
		cancel()
		listErr = err
		if err == nil {
			log.Println("Qdrantサーバーの準備ができました。")
			for _, collection := range res.GetCollections() {
				if collection.GetName() == collectionName {
					collectionExists = true
					break
				}
			}
			break // 成功したのでループを抜ける
		}
		log.Printf("Qdrantサーバーの準備確認に失敗しました (試行 %d/%d)。%v後に再試行します...", i+1, maxRetries, retryInterval)
		time.Sleep(retryInterval)
	}

	if listErr != nil {
		log.Fatalf("Qdrantのコレクションリスト取得に失敗しました（リトライ上限到達）: %v", listErr)
	}

	// コレクションが存在しない場合は作成
	if !collectionExists {
		log.Printf("コレクション '%s' が存在しないため、新規作成します。", collectionName)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err = qdrantCollectionsClient.Create(ctx, &qdrant.CreateCollection{
			CollectionName: collectionName,
			VectorsConfig: &qdrant.VectorsConfig{
				Config: &qdrant.VectorsConfig_Params{
					Params: &qdrant.VectorParams{
						Size:     vectorSize,
						Distance: qdrant.Distance_Cosine,
					},
				},
			},
		})
		if err != nil {
			log.Fatalf("Qdrantのコレクション作成に失敗しました: %v", err)
		}
		log.Printf("コレクション '%s' を作成しました。", collectionName)
	} else {
		log.Printf("コレクション '%s' は既に存在します。", collectionName)
	}

	return &VectorStoreService{
		qdrantClient:            qdrantPointsClient,
		qdrantCollectionsClient: qdrantCollectionsClient,
		azureOpenAIService:      azureOpenAIService,
	}
}

// Save はテキストをベクトル化し、メタデータと共にQdrantに保存します。
func (s *VectorStoreService) Save(ctx context.Context, text string, metadata map[string]interface{}) error {
	// 1. テキストをベクトル化
	vector, err := s.azureOpenAIService.CreateEmbedding(ctx, text)
	if err != nil {
		return fmt.Errorf("テキストのベクトル化に失敗: %w", err)
	}

	// 2. Qdrantのペイロードを作成
	payload := make(map[string]*qdrant.Value)
	for key, value := range metadata {
		// 型スイッチでqdrant.Valueに変換
		switch v := value.(type) {
		case string:
			payload[key] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: v}}
		case int:
			payload[key] = &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(v)}}
		case float64:
			payload[key] = &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: v}}
		case bool:
			payload[key] = &qdrant.Value{Kind: &qdrant.Value_BoolValue{BoolValue: v}}
		}
	}
	// 元のテキストもペイロードに含める
	payload["text"] = &qdrant.Value{
		Kind: &qdrant.Value_StringValue{StringValue: text},
	}

	// 3. Qdrantに保存するPointを作成
	pointId := uuid.New().String()
	points := []*qdrant.PointStruct{
		{
			Id: &qdrant.PointId{
				PointIdOptions: &qdrant.PointId_Uuid{Uuid: pointId},
			},
			Vectors: &qdrant.Vectors{
				VectorsOptions: &qdrant.Vectors_Vector{
					Vector: &qdrant.Vector{
						Data: vector,
					},
				},
			},
			Payload: payload,
		},
	}

	// 4. QdrantにUpsert
	collectionName := "hunt_chat_documents"
	waitUpsert := true
	_, err = s.qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points:         points,
		Wait:           &waitUpsert,
	})

	if err != nil {
		return fmt.Errorf("Qdrantへのベクトル保存に失敗: %w", err)
	}

	log.Printf("ID '%s' のベクトルをQdrantに保存しました。", pointId)
	return nil
}

// Search はクエリテキストに類似したベクトルをQdrantから検索します。
func (s *VectorStoreService) Search(ctx context.Context, queryText string, topK uint64) ([]*qdrant.ScoredPoint, error) {
	// 1. クエリテキストをベクトル化
	queryVector, err := s.azureOpenAIService.CreateEmbedding(ctx, queryText)
	if err != nil {
		return nil, fmt.Errorf("クエリテキストのベクトル化に失敗: %w", err)
	}

	// 2. Qdrantで類似ベクトルを検索
	collectionName := "hunt_chat_documents"
	withPayload := true
	searchResult, err := s.qdrantClient.Search(ctx, &qdrant.SearchPoints{
		CollectionName: collectionName,
		Vector:         queryVector,
		Limit:          topK,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: withPayload}},
	})
	if err != nil {
		return nil, fmt.Errorf("Qdrantでのベクトル検索に失敗: %w", err)
	}

	log.Printf("'%s' に類似した %d 件の結果をQdrantから取得しました。", queryText, len(searchResult.GetResult()))
	return searchResult.GetResult(), nil
}

// SaveAnalysisReport 分析レポートを構造化してQdrantに保存
func (s *VectorStoreService) SaveAnalysisReport(ctx context.Context, report interface{}, reportType string) error {
	// レポートをJSON文字列に変換
	var reportText string
	switch r := report.(type) {
	case string:
		reportText = r
	default:
		// 構造体の場合はフォーマットして保存
		reportText = fmt.Sprintf("%+v", r)
	}

	metadata := map[string]interface{}{
		"type":        "analysis_report",
		"report_type": reportType,
		"timestamp":   time.Now().Format(time.RFC3339),
		"source":      "statistical_analysis",
	}

	return s.Save(ctx, reportText, metadata)
}

// SearchAnalysisReports 分析レポートを検索（typeフィルタ付き）
func (s *VectorStoreService) SearchAnalysisReports(ctx context.Context, query string, topK uint64) ([]*qdrant.ScoredPoint, error) {
	// クエリテキストをベクトル化
	queryVector, err := s.azureOpenAIService.CreateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("クエリテキストのベクトル化に失敗: %w", err)
	}

	// typeフィルタを追加
	collectionName := "hunt_chat_documents"
	withPayload := true

	// Qdrantのフィルタ条件を構築
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "type",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: "analysis_report",
							},
						},
					},
				},
			},
		},
	}

	searchResult, err := s.qdrantClient.Search(ctx, &qdrant.SearchPoints{
		CollectionName: collectionName,
		Vector:         queryVector,
		Limit:          topK,
		Filter:         filter,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: withPayload}},
	})
	if err != nil {
		return nil, fmt.Errorf("分析レポートの検索に失敗: %w", err)
	}

	log.Printf("分析レポート検索: '%s' に類似した %d 件を取得", query, len(searchResult.GetResult()))
	return searchResult.GetResult(), nil
}

// StoreDocument は指定されたコレクションにドキュメントを保存（コレクション名を指定可能）
func (s *VectorStoreService) StoreDocument(ctx context.Context, collectionName string, documentID string, text string, metadata map[string]interface{}) error {
	// コレクションが存在するか確認し、なければ作成
	if err := s.ensureCollection(ctx, collectionName); err != nil {
		return fmt.Errorf("コレクションの準備に失敗: %w", err)
	}

	// 1. テキストをベクトル化
	vector, err := s.azureOpenAIService.CreateEmbedding(ctx, text)
	if err != nil {
		return fmt.Errorf("テキストのベクトル化に失敗: %w", err)
	}

	// 2. Qdrantのペイロードを作成
	payload := make(map[string]*qdrant.Value)
	for key, value := range metadata {
		switch v := value.(type) {
		case string:
			payload[key] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: v}}
		case int:
			payload[key] = &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(v)}}
		case int64:
			payload[key] = &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: v}}
		case float64:
			payload[key] = &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: v}}
		case bool:
			payload[key] = &qdrant.Value{Kind: &qdrant.Value_BoolValue{BoolValue: v}}
		}
	}
	// 元のテキストもペイロードに含める
	payload["text"] = &qdrant.Value{
		Kind: &qdrant.Value_StringValue{StringValue: text},
	}

	// 3. Qdrantに保存するPointを作成
	points := []*qdrant.PointStruct{
		{
			Id: &qdrant.PointId{
				PointIdOptions: &qdrant.PointId_Uuid{Uuid: documentID},
			},
			Vectors: &qdrant.Vectors{
				VectorsOptions: &qdrant.Vectors_Vector{
					Vector: &qdrant.Vector{
						Data: vector,
					},
				},
			},
			Payload: payload,
		},
	}

	// 4. QdrantにUpsert
	waitUpsert := true
	_, err = s.qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points:         points,
		Wait:           &waitUpsert,
	})

	if err != nil {
		return fmt.Errorf("Qdrantへのドキュメント保存に失敗: %w", err)
	}

	log.Printf("ドキュメント '%s' をコレクション '%s' に保存しました。", documentID, collectionName)
	return nil
}

// SearchWithFilter はフィルタ条件付きで検索
func (s *VectorStoreService) SearchWithFilter(ctx context.Context, collectionName string, queryText string, topK uint64, filter *qdrant.Filter) ([]*qdrant.ScoredPoint, error) {
	// コレクションの存在を確認
	if err := s.ensureCollection(ctx, collectionName); err != nil {
		return nil, fmt.Errorf("コレクションの確認に失敗: %w", err)
	}

	// 1. クエリテキストをベクトル化
	queryVector, err := s.azureOpenAIService.CreateEmbedding(ctx, queryText)
	if err != nil {
		return nil, fmt.Errorf("クエリテキストのベクトル化に失敗: %w", err)
	}

	// 2. Qdrantで類似ベクトルを検索（フィルタ付き）
	withPayload := true
	searchResult, err := s.qdrantClient.Search(ctx, &qdrant.SearchPoints{
		CollectionName: collectionName,
		Vector:         queryVector,
		Limit:          topK,
		Filter:         filter,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: withPayload}},
	})
	if err != nil {
		return nil, fmt.Errorf("Qdrantでのフィルタ付き検索に失敗: %w", err)
	}

	log.Printf("コレクション '%s' でフィルタ付き検索: %d 件取得", collectionName, len(searchResult.GetResult()))
	return searchResult.GetResult(), nil
}

// ensureCollection コレクションが存在することを確認し、なければ作成
func (s *VectorStoreService) ensureCollection(ctx context.Context, collectionName string) error {
	log.Printf("コレクション '%s' の存在を確認中...", collectionName)

	// コレクションのリストを取得
	listCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := s.qdrantCollectionsClient.List(listCtx, &qdrant.ListCollectionsRequest{})
	if err != nil {
		log.Printf("警告: コレクションリストの取得に失敗（続行します）: %v", err)
		return nil // エラーでも続行（既存の場合はUpsert時に成功する）
	}

	// コレクションが存在するか確認
	collectionExists := false
	for _, collection := range res.GetCollections() {
		if collection.GetName() == collectionName {
			collectionExists = true
			break
		}
	}

	// 存在しない場合は作成
	if !collectionExists {
		log.Printf("コレクション '%s' を作成します...", collectionName)
		createCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		vectorSize := uint64(1536) // text-embedding-3-smallの次元数
		_, err = s.qdrantCollectionsClient.Create(createCtx, &qdrant.CreateCollection{
			CollectionName: collectionName,
			VectorsConfig: &qdrant.VectorsConfig{
				Config: &qdrant.VectorsConfig_Params{
					Params: &qdrant.VectorParams{
						Size:     vectorSize,
						Distance: qdrant.Distance_Cosine,
					},
				},
			},
		})
		if err != nil {
			log.Printf("警告: コレクション作成に失敗（続行します）: %v", err)
			return nil // エラーでも続行
		}
		log.Printf("コレクション '%s' を作成しました", collectionName)
	} else {
		log.Printf("コレクション '%s' は既に存在します", collectionName)
	}

	return nil
}

// SaveChatHistory チャット履歴をQdrantに保存
func (s *VectorStoreService) SaveChatHistory(ctx context.Context, entry models.ChatHistoryEntry) error {
	collectionName := "chat_history"

	// エントリーをJSON文字列に変換してテキストベクトル化用に準備
	entryText := fmt.Sprintf(
		"Role: %s\nMessage: %s\nContext: %s\nTags: %v\nIntent: %s\nProductID: %s",
		entry.Role,
		entry.Message,
		entry.Context,
		entry.Tags,
		entry.Metadata.Intent,
		entry.Metadata.ProductID,
	)

	// メタデータを準備
	metadata := map[string]interface{}{
		"type":       "chat_history",
		"session_id": entry.SessionID,
		"user_id":    entry.UserID,
		"role":       entry.Role,
		"timestamp":  entry.Timestamp,
		"intent":     entry.Metadata.Intent,
		"product_id": entry.Metadata.ProductID,
		"date_range": entry.Metadata.DateRange,
	}

	// タグをJSON文字列として追加
	if len(entry.Tags) > 0 {
		tagsJSON, _ := json.Marshal(entry.Tags)
		metadata["tags"] = string(tagsJSON)
	}

	// キーワードをJSON文字列として追加
	if len(entry.Metadata.TopicKeywords) > 0 {
		keywordsJSON, _ := json.Marshal(entry.Metadata.TopicKeywords)
		metadata["keywords"] = string(keywordsJSON)
	}

	// ドキュメントとして保存（既存のStoreDocumentメソッドを活用）
	return s.StoreDocument(ctx, collectionName, entry.ID, entryText, metadata)
}

// SearchChatHistory チャット履歴を検索（RAG機能）
func (s *VectorStoreService) SearchChatHistory(ctx context.Context, query string, sessionID string, userID string, topK uint64) ([]models.ChatHistoryEntry, error) {
	collectionName := "chat_history"

	// フィルタ条件を構築
	var filterConditions []*qdrant.Condition

	// typeフィルタは必須
	filterConditions = append(filterConditions, &qdrant.Condition{
		ConditionOneOf: &qdrant.Condition_Field{
			Field: &qdrant.FieldCondition{
				Key: "type",
				Match: &qdrant.Match{
					MatchValue: &qdrant.Match_Keyword{
						Keyword: "chat_history",
					},
				},
			},
		},
	})

	// セッションIDフィルタ（オプション）
	if sessionID != "" {
		filterConditions = append(filterConditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key: "session_id",
					Match: &qdrant.Match{
						MatchValue: &qdrant.Match_Keyword{
							Keyword: sessionID,
						},
					},
				},
			},
		})
	}

	// ユーザーIDフィルタ（オプション）
	if userID != "" {
		filterConditions = append(filterConditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key: "user_id",
					Match: &qdrant.Match{
						MatchValue: &qdrant.Match_Keyword{
							Keyword: userID,
						},
					},
				},
			},
		})
	}

	filter := &qdrant.Filter{
		Must: filterConditions,
	}

	// ベクトル検索を実行
	results, err := s.SearchWithFilter(ctx, collectionName, query, topK, filter)
	if err != nil {
		return nil, fmt.Errorf("チャット履歴の検索に失敗: %w", err)
	}

	// 結果を ChatHistoryEntry に変換
	var entries []models.ChatHistoryEntry
	for _, result := range results {
		payload := result.GetPayload()

		entry := models.ChatHistoryEntry{
			ID:        result.Id.GetUuid(),
			SessionID: getStringFromPayload(payload, "session_id"),
			UserID:    getStringFromPayload(payload, "user_id"),
			Role:      getStringFromPayload(payload, "role"),
			Message:   getStringFromPayload(payload, "text"), // 元のテキストフィールドから取得
			Timestamp: getStringFromPayload(payload, "timestamp"),
			Metadata: models.Metadata{
				Intent:         getStringFromPayload(payload, "intent"),
				ProductID:      getStringFromPayload(payload, "product_id"),
				DateRange:      getStringFromPayload(payload, "date_range"),
				RelevanceScore: float64(result.GetScore()),
			},
		}

		// タグの復元
		if tagsJSON := getStringFromPayload(payload, "tags"); tagsJSON != "" {
			var tags []string
			if err := json.Unmarshal([]byte(tagsJSON), &tags); err == nil {
				entry.Tags = tags
			}
		}

		// キーワードの復元
		if keywordsJSON := getStringFromPayload(payload, "keywords"); keywordsJSON != "" {
			var keywords []string
			if err := json.Unmarshal([]byte(keywordsJSON), &keywords); err == nil {
				entry.Metadata.TopicKeywords = keywords
			}
		}

		entries = append(entries, entry)
	}

	log.Printf("チャット履歴検索: %d 件の関連する会話を取得しました", len(entries))
	return entries, nil
}

// GetRecentChatHistory 最近のチャット履歴を取得（時系列順）
func (s *VectorStoreService) GetRecentChatHistory(ctx context.Context, sessionID string, limit int) ([]models.ChatHistoryEntry, error) {
	// この機能は、Qdrantではスコアベースの検索になるため、
	// 実際には時系列での取得が難しい。代わりにベクトル検索で直近の会話を取得する
	// 改善案: timestampを使った専用のフィルタリング機能を追加する
	return s.SearchChatHistory(ctx, "最近の会話", sessionID, "", uint64(limit))
}

// getStringFromPayload ペイロードから文字列値を取得するヘルパー関数
func getStringFromPayload(payload map[string]*qdrant.Value, key string) string {
	if val, ok := payload[key]; ok {
		if strVal := val.GetStringValue(); strVal != "" {
			return strVal
		}
	}
	return ""
}
