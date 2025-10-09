package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// VectorStoreService はQdrantとのやり取りを管理します
type VectorStoreService struct {
	qdrantClient       qdrant.PointsClient
	azureOpenAIService *AzureOpenAIService
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
		qdrantClient:       qdrantPointsClient,
		azureOpenAIService: azureOpenAIService,
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
