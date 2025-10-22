package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"sort"
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
func NewVectorStoreService(azureOpenAIService *AzureOpenAIService, qdrantURL string, qdrantAPIKey string) (*VectorStoreService, error) {
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
		return nil, fmt.Errorf("QdrantへのgRPCクライアント作成に失敗しました: %w", err)
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
		return nil, fmt.Errorf("Qdrantのコレクションリスト取得に失敗しました（リトライ上限到達）: %w", listErr)
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
			return nil, fmt.Errorf("Qdrantのコレクション作成に失敗しました: %w", err)
		}
		log.Printf("コレクション '%s' を作成しました。", collectionName)
	} else {
		log.Printf("コレクション '%s' は既に存在します。", collectionName)
	}

	return &VectorStoreService{
		qdrantClient:            qdrantPointsClient,
		qdrantCollectionsClient: qdrantCollectionsClient,
		azureOpenAIService:      azureOpenAIService,
	}, nil
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

// GetAllAnalysisReportHeaders はすべての分析レポートのヘッダー情報を取得します
func (s *VectorStoreService) GetAllAnalysisReportHeaders(ctx context.Context) ([]models.AnalysisReportHeader, error) {
	collectionName := "hunt_chat_documents"
	points, err := s.ScrollAllPoints(ctx, collectionName, 1000) // 最大1000件まで取得
	if err != nil {
		return nil, fmt.Errorf("レポートの取得に失敗: %w", err)
	}

	var headers []models.AnalysisReportHeader
	for _, point := range points {
		if point.Payload != nil && point.Payload["type"] != nil && point.Payload["type"].GetStringValue() == "analysis_report" {
			headers = append(headers, models.AnalysisReportHeader{
				ReportID:     point.Id.GetUuid(),
				FileName:     getStringFromPayload(point.Payload, "file_name"),
				AnalysisDate: getStringFromPayload(point.Payload, "analysis_date"),
				// DateRangeはペイロードにないので、必要であれば別途追加する
			})
		}
	}

	// 日付の降順（新しいものが先）にソート
	sort.Slice(headers, func(i, j int) bool {
		t1, _ := time.Parse(time.RFC3339, headers[i].AnalysisDate)
		t2, _ := time.Parse(time.RFC3339, headers[j].AnalysisDate)
		return t1.After(t2)
	})

	log.Printf("%d件の分析レポートヘッダーを取得しました", len(headers))
	return headers, nil
}

// GetAllAnalysisReports はすべての分析レポートを完全に取得します
func (s *VectorStoreService) GetAllAnalysisReports(ctx context.Context) ([]models.AnalysisReport, error) {
	collectionName := "hunt_chat_documents"
	points, err := s.ScrollAllPoints(ctx, collectionName, 1000) // 最大1000件まで取得
	if err != nil {
		return nil, fmt.Errorf("全レポートの取得に失敗: %w", err)
	}

	var reports []models.AnalysisReport
	for _, point := range points {
		if point.Payload != nil && point.Payload["type"] != nil && point.Payload["type"].GetStringValue() == "analysis_report" {
			// ★ textからではなく、full_report_jsonから取得する
			reportJSON := getStringFromPayload(point.Payload, "full_report_json")
			if reportJSON == "" {
				log.Printf("レポートID %s に full_report_json が見つかりません。スキップします。", point.Id.GetUuid())
				continue
			}
			var report models.AnalysisReport
			if err := json.Unmarshal([]byte(reportJSON), &report); err == nil {
				reports = append(reports, report)
			} else {
				log.Printf("レポートJSONのパースに失敗 (GetAllAnalysisReports): %v", err)
			}
		}
	}

	log.Printf("%d件の完全な分析レポートを取得しました", len(reports))
	return reports, nil
}

// GetAllAnomalyResponses はすべての異常回答を取得します
func (s *VectorStoreService) GetAllAnomalyResponses(ctx context.Context) ([]models.AnomalyResponse, error) {
	collectionName := "anomaly_responses"
	points, err := s.ScrollAllPoints(ctx, collectionName, 10000) // 十分な数を取得
	if err != nil {
		return nil, fmt.Errorf("全異常回答の取得に失敗: %w", err)
	}

	var responses []models.AnomalyResponse
	for _, point := range points {
		if point.Payload == nil {
			continue
		}

		// textペイロードから回答のJSON文字列を取得しパースする
		// SaveAnomalyResponseの実装に合わせて修正
		responseText := getStringFromPayload(point.Payload, "text")
		var response models.AnomalyResponse

		// ペイロードから直接フィールドを読み込む方が確実
		response.ResponseID = getStringFromPayload(point.Payload, "response_id")
		response.AnomalyDate = getStringFromPayload(point.Payload, "anomaly_date")
		response.ProductID = getStringFromPayload(point.Payload, "product_id")
		response.Question = getStringFromPayload(point.Payload, "question")
		response.Answer = getStringFromPayload(point.Payload, "answer")

		if response.AnomalyDate != "" && response.ProductID != "" {
			responses = append(responses, response)
		} else if responseText != "" {
			// textからのパースも試みる（後方互換性のため）
			// この部分はSaveAnomalyResponseの実装に依存します
			// 現在の実装ではtextにJSONは入っていないため、ペイロードからの読み込みがメイン
			continue
		}
	}

	log.Printf("%d件の異常回答を取得しました", len(responses))
	return responses, nil
}

// GetAnalysisReportByID はIDで単一の分析レポートを取得します
func (s *VectorStoreService) GetAnalysisReportByID(ctx context.Context, reportID string) (*models.AnalysisReport, error) {
	collectionName := "hunt_chat_documents"

	// IDでフィルタリングしてScrollで取得
	scrollResult, err := s.qdrantClient.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: collectionName,
		Filter: &qdrant.Filter{
			Must: []*qdrant.Condition{
				{
					ConditionOneOf: &qdrant.Condition_HasId{
						HasId: &qdrant.HasIdCondition{
							HasId: []*qdrant.PointId{
								{PointIdOptions: &qdrant.PointId_Uuid{Uuid: reportID}},
							},
						},
					},
				},
			},
		},
		Limit:       func(u uint32) *uint32 { return &u }(1),
		WithPayload: &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})

	if err != nil {
		return nil, fmt.Errorf("Qdrantからのポイント取得に失敗: %w", err)
	}

	if len(scrollResult.GetResult()) == 0 {
		return nil, fmt.Errorf("レポートID '%s' が見つかりません", reportID)
	}

	point := scrollResult.GetResult()[0]
	if point.Payload == nil || point.Payload["type"] == nil || point.Payload["type"].GetStringValue() != "analysis_report" {
		return nil, fmt.Errorf("ポイント '%s' は分析レポートではありません", reportID)
	}

	// ★ textからではなく、full_report_jsonから取得する
	reportJSON := getStringFromPayload(point.Payload, "full_report_json")
	if reportJSON == "" {
		return nil, fmt.Errorf("レポートID '%s' に full_report_json が見つかりません", reportID)
	}

	var report models.AnalysisReport
	if err := json.Unmarshal([]byte(reportJSON), &report); err != nil {
		return nil, fmt.Errorf("レポートJSONのパースに失敗: %w", err)
	}

	log.Printf("レポート '%s' を取得しました", reportID)
	return &report, nil
}

// DeleteAllAnalysisReports はすべての分析レポートを削除します
func (s *VectorStoreService) DeleteAllAnalysisReports(ctx context.Context) error {
	collectionName := "hunt_chat_documents"
	points, err := s.ScrollAllPoints(ctx, collectionName, 10000) // Adjust limit as needed
	if err != nil {
		return fmt.Errorf("レポートの取得に失敗: %w", err)
	}

	var idsToDelete []*qdrant.PointId
	for _, point := range points {
		if point.Payload == nil {
			continue
		}
		if payloadType, ok := point.Payload["type"]; ok && payloadType != nil {
			if payloadType.GetStringValue() == "analysis_report" {
				idsToDelete = append(idsToDelete, point.Id)
			}
		}
	}

	if len(idsToDelete) == 0 {
		log.Println("削除対象の分析レポートはありませんでした")
		return nil
	}

	waitDelete := true
	_, err = s.qdrantClient.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: collectionName,
		Wait:           &waitDelete,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{Ids: idsToDelete},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("Qdrantからの分析レポート削除に失敗: %w", err)
	}

	log.Printf("%d件の分析レポートを削除しました", len(idsToDelete))
	return nil
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

// ScrollAllPoints 指定したコレクションの全ポイントを取得（フィルタなし）
func (s *VectorStoreService) ScrollAllPoints(ctx context.Context, collectionName string, limit uint32) ([]*qdrant.RetrievedPoint, error) {
	// コレクションの存在を確認
	if err := s.ensureCollection(ctx, collectionName); err != nil {
		return nil, fmt.Errorf("コレクションの確認に失敗: %w", err)
	}

	withPayload := true
	scrollResult, err := s.qdrantClient.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: collectionName,
		Limit:          &limit,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: withPayload}},
	})

	if err != nil {
		return nil, fmt.Errorf("Qdrantでの全件取得に失敗: %w", err)
	}

	log.Printf("コレクション '%s' から %d 件取得", collectionName, len(scrollResult.GetResult()))
	return scrollResult.GetResult(), nil
}

// DeletePoint 指定したIDのポイントを削除
func (s *VectorStoreService) DeletePoint(ctx context.Context, collectionName string, pointID string) error {
	// コレクションの存在を確認
	if err := s.ensureCollection(ctx, collectionName); err != nil {
		return fmt.Errorf("コレクションの確認に失敗: %w", err)
	}

	waitDelete := true
	_, err := s.qdrantClient.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: collectionName,
		Wait:           &waitDelete,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{
						{
							PointIdOptions: &qdrant.PointId_Uuid{
								Uuid: pointID,
							},
						},
					},
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("Qdrantからのポイント削除に失敗: %w", err)
	}

	log.Printf("ポイント '%s' をコレクション '%s' から削除しました", pointID, collectionName)
	return nil
}

// DeleteDocumentByFileName は指定したfile_nameを持つ全ポイントを削除
func (s *VectorStoreService) DeleteDocumentByFileName(ctx context.Context, collectionName string, fileName string) error {
	// コレクションの存在確認
	if err := s.ensureCollection(ctx, collectionName); err != nil {
		return fmt.Errorf("コレクションの確認に失敗: %w", err)
	}

	// file_nameでフィルタして全ポイントを取得
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "file_name",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: fileName,
							},
						},
					},
				},
			},
		},
	}

	withPayload := false
	limit := uint32(1000) // 最大1000チャンク
	scrollResult, err := s.qdrantClient.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: collectionName,
		Filter:         filter,
		Limit:          &limit,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: withPayload}},
	})
	if err != nil {
		return fmt.Errorf("ポイント取得に失敗: %w", err)
	}

	points := scrollResult.GetResult()
	if len(points) == 0 {
		// 初回実行時は削除対象がないので、これはエラーではない
		return nil
	}

	// ポイントIDを収集
	var idsToDelete []*qdrant.PointId
	for _, point := range points {
		idsToDelete = append(idsToDelete, point.Id)
	}

	// 削除実行
	waitDelete := true
	_, err = s.qdrantClient.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: collectionName,
		Wait:           &waitDelete,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{Ids: idsToDelete},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("ポイント削除に失敗: %w", err)
	}

	log.Printf("  🗑️ %d 個の古いチャンクを削除しました", len(idsToDelete))
	return nil
}

// EnsureEconomicCollection ensures the economic_daily_summaries collection exists and indexes are set.
func (s *VectorStoreService) EnsureEconomicCollection(ctx context.Context) error {
	collectionName := "economic_daily_summaries"
	if err := s.ensureCollection(ctx, collectionName); err != nil {
		return err
	}
	// Create indexes on symbol and date for efficient filtering
	fieldType := qdrant.FieldType_FieldTypeKeyword
	// symbol
	idxCtx, cancelIdx := context.WithTimeout(ctx, 10*time.Second)
	defer cancelIdx()
	_, err := s.qdrantClient.CreateFieldIndex(idxCtx, &qdrant.CreateFieldIndexCollection{
		CollectionName: collectionName,
		FieldName:      "symbol",
		FieldType:      &fieldType,
	})
	if err != nil {
		log.Printf("ℹ️ symbol index create (maybe exists): %v", err)
	}
	// date
	idxCtx2, cancelIdx2 := context.WithTimeout(ctx, 10*time.Second)
	defer cancelIdx2()
	_, err = s.qdrantClient.CreateFieldIndex(idxCtx2, &qdrant.CreateFieldIndexCollection{
		CollectionName: collectionName,
		FieldName:      "date",
		FieldType:      &fieldType,
	})
	if err != nil {
		log.Printf("ℹ️ date index create (maybe exists): %v", err)
	}
	return nil
}

// GetLatestEconomicDate returns the max date (YYYY-MM-DD) stored for a symbol.
func (s *VectorStoreService) GetLatestEconomicDate(ctx context.Context, symbol string) (string, error) {
	collectionName := "economic_daily_summaries"
	if err := s.ensureCollection(ctx, collectionName); err != nil {
		return "", err
	}

	// Build filter: type == economic_daily AND symbol == symbol
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{Field: &qdrant.FieldCondition{
					Key:   "type",
					Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: "economic_daily"}},
				}},
			},
			{
				ConditionOneOf: &qdrant.Condition_Field{Field: &qdrant.FieldCondition{
					Key:   "symbol",
					Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: symbol}},
				}},
			},
		},
	}

	limit := uint32(10000)
	withPayload := true
	scrollCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	res, err := s.qdrantClient.Scroll(scrollCtx, &qdrant.ScrollPoints{
		CollectionName: collectionName,
		Filter:         filter,
		Limit:          &limit,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: withPayload}},
	})
	if err != nil {
		return "", fmt.Errorf("failed to scroll economic points: %w", err)
	}
	latest := ""
	for _, p := range res.GetResult() {
		d := getStringFromPayload(p.Payload, "date")
		if d > latest { // lexicographical works for YYYY-MM-DD
			latest = d
		}
	}
	return latest, nil
}

// GetEconomicSeries fetches economic daily series for a symbol between start and end (inclusive), sorted by date asc.
func (s *VectorStoreService) GetEconomicSeries(ctx context.Context, symbol string, start, end time.Time) ([]struct{ Date string; Value float64 }, error) {
	collectionName := "economic_daily_summaries"
	if err := s.ensureCollection(ctx, collectionName); err != nil {
		return nil, err
	}

	// Filter by type and symbol; we'll post-filter dates client-side
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{ConditionOneOf: &qdrant.Condition_Field{Field: &qdrant.FieldCondition{Key: "type", Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: "economic_daily"}}}}},
			{ConditionOneOf: &qdrant.Condition_Field{Field: &qdrant.FieldCondition{Key: "symbol", Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: symbol}}}}},
		},
	}
	limit := uint32(100000)
	withPayload := true
	scrollCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	res, err := s.qdrantClient.Scroll(scrollCtx, &qdrant.ScrollPoints{
		CollectionName: collectionName,
		Filter:         filter,
		Limit:          &limit,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: withPayload}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scroll economic series: %w", err)
	}
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")
	out := make([]struct{ Date string; Value float64 }, 0, len(res.GetResult()))
	for _, p := range res.GetResult() {
		if p.Payload == nil { continue }
		d := getStringFromPayload(p.Payload, "date")
		if d == "" { continue }
		if d < startStr || d > endStr { continue }
		var val float64
		if v := p.Payload["value"]; v != nil {
			val = v.GetDoubleValue()
		}
		out = append(out, struct{ Date string; Value float64 }{Date: d, Value: val})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out, nil
}

// StoreEconomicDailyBatch stores daily economic points for a symbol, embedding textual summaries.
func (s *VectorStoreService) StoreEconomicDailyBatch(ctx context.Context, symbol string, points []struct {
	Date  string
	Value float64
}) error {
	collectionName := "economic_daily_summaries"
	// Ensure collection with a short timeout
	ensureCtx, cancelEnsure := context.WithTimeout(ctx, 15*time.Second)
	defer cancelEnsure()
	if err := s.EnsureEconomicCollection(ensureCtx); err != nil {
		return err
	}

	// Build points
	qpoints := make([]*qdrant.PointStruct, 0, len(points))
	// Economic daily summaries don't need semantic search now; avoid slow embeddings.
	// Use a fixed zero vector with the known dimension (1536) to satisfy Qdrant schema.
	zeroVec := make([]float32, 1536)
	for _, pt := range points {
		text := fmt.Sprintf("%s %s Close=%.4f", pt.Date, symbol, pt.Value)
		payload := map[string]*qdrant.Value{
			"type":   {Kind: &qdrant.Value_StringValue{StringValue: "economic_daily"}},
			"symbol": {Kind: &qdrant.Value_StringValue{StringValue: symbol}},
			"date":   {Kind: &qdrant.Value_StringValue{StringValue: pt.Date}},
			"value":  {Kind: &qdrant.Value_DoubleValue{DoubleValue: pt.Value}},
			"text":   {Kind: &qdrant.Value_StringValue{StringValue: text}},
		}
		// use deterministic UUID (v5/SHA1) derived from "symbol:date" for idempotency
		rawID := fmt.Sprintf("%s:%s", symbol, pt.Date)
		idStr := uuid.NewSHA1(uuid.NameSpaceURL, []byte(rawID)).String()
		qpoints = append(qpoints, &qdrant.PointStruct{
			Id:      &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: idStr}},
			Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: zeroVec}}},
			Payload: payload,
		})
	}
	if len(qpoints) == 0 {
		return nil
	}
	wait := true
	// Bound upsert to avoid long hangs
	upsertCtx, cancelUpsert := context.WithTimeout(ctx, 20*time.Second)
	defer cancelUpsert()
	_, err := s.qdrantClient.Upsert(upsertCtx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points:         qpoints,
		Wait:           &wait,
	})
	if err != nil {
		return fmt.Errorf("upsert economic points failed: %w", err)
	}
	log.Printf("✅ Stored %d economic points for %s", len(qpoints), symbol)
	return nil
}

// EnsureSalesCollection ensures the sales_daily_series collection exists and indexes are set.
func (s *VectorStoreService) EnsureSalesCollection(ctx context.Context) error {
	collectionName := "sales_daily_series"
	if err := s.ensureCollection(ctx, collectionName); err != nil {
		return err
	}
	fieldType := qdrant.FieldType_FieldTypeKeyword
	// product_id
	ctx1, cancel1 := context.WithTimeout(ctx, 10*time.Second)
	defer cancel1()
	if _, err := s.qdrantClient.CreateFieldIndex(ctx1, &qdrant.CreateFieldIndexCollection{
		CollectionName: collectionName,
		FieldName:      "product_id",
		FieldType:      &fieldType,
	}); err != nil {
		log.Printf("ℹ️ product_id index create (maybe exists): %v", err)
	}
	// date
	ctx2, cancel2 := context.WithTimeout(ctx, 10*time.Second)
	defer cancel2()
	if _, err := s.qdrantClient.CreateFieldIndex(ctx2, &qdrant.CreateFieldIndexCollection{
		CollectionName: collectionName,
		FieldName:      "date",
		FieldType:      &fieldType,
	}); err != nil {
		log.Printf("ℹ️ date index create (maybe exists): %v", err)
	}
	return nil
}

// GetLatestSalesDate returns the max date (YYYY-MM-DD) stored for a product.
func (s *VectorStoreService) GetLatestSalesDate(ctx context.Context, productID string) (string, error) {
	collectionName := "sales_daily_series"
	if err := s.ensureCollection(ctx, collectionName); err != nil {
		return "", err
	}
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{ConditionOneOf: &qdrant.Condition_Field{Field: &qdrant.FieldCondition{Key: "type", Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: "sales_daily"}}}}},
			{ConditionOneOf: &qdrant.Condition_Field{Field: &qdrant.FieldCondition{Key: "product_id", Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: productID}}}}},
		},
	}
	limit := uint32(100000)
	withPayload := true
	ctxScroll, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	res, err := s.qdrantClient.Scroll(ctxScroll, &qdrant.ScrollPoints{
		CollectionName: collectionName,
		Filter:         filter,
		Limit:          &limit,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: withPayload}},
	})
	if err != nil {
		return "", fmt.Errorf("failed to scroll sales points: %w", err)
	}
	latest := ""
	for _, p := range res.GetResult() {
		d := getStringFromPayload(p.Payload, "date")
		if d > latest { latest = d }
	}
	return latest, nil
}

// StoreSalesDailyBatch stores daily sales points for a product.
func (s *VectorStoreService) StoreSalesDailyBatch(ctx context.Context, productID string, points []struct{ Date string; Sales float64 }) error {
	collectionName := "sales_daily_series"
	// Ensure collection with timeout
	ectx, cancelE := context.WithTimeout(ctx, 15*time.Second)
	defer cancelE()
	if err := s.EnsureSalesCollection(ectx); err != nil { return err }

	// Prepare points with zero vector
	zeroVec := make([]float32, 1536)
	qpoints := make([]*qdrant.PointStruct, 0, len(points))
	for _, pt := range points {
		text := fmt.Sprintf("%s %s Sales=%.4f", pt.Date, productID, pt.Sales)
		payload := map[string]*qdrant.Value{
			"type":       {Kind: &qdrant.Value_StringValue{StringValue: "sales_daily"}},
			"product_id": {Kind: &qdrant.Value_StringValue{StringValue: productID}},
			"date":       {Kind: &qdrant.Value_StringValue{StringValue: pt.Date}},
			"sales":      {Kind: &qdrant.Value_DoubleValue{DoubleValue: pt.Sales}},
			"text":       {Kind: &qdrant.Value_StringValue{StringValue: text}},
		}
		rawID := fmt.Sprintf("%s:%s", productID, pt.Date)
		idStr := uuid.NewSHA1(uuid.NameSpaceURL, []byte(rawID)).String()
		qpoints = append(qpoints, &qdrant.PointStruct{
			Id:      &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: idStr}},
			Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: zeroVec}}},
			Payload: payload,
		})
	}
	if len(qpoints) == 0 { return nil }
	wait := true
	uctx, cancelU := context.WithTimeout(ctx, 20*time.Second)
	defer cancelU()
	if _, err := s.qdrantClient.Upsert(uctx, &qdrant.UpsertPoints{CollectionName: collectionName, Points: qpoints, Wait: &wait}); err != nil {
		return fmt.Errorf("upsert sales points failed: %w", err)
	}
	log.Printf("✅ Stored %d sales points for %s", len(qpoints), productID)
	return nil
}

// GetSalesSeries fetches sales daily series for a product between start and end.
func (s *VectorStoreService) GetSalesSeries(ctx context.Context, productID string, start, end time.Time) ([]struct{ Date string; Sales float64 }, error) {
	collectionName := "sales_daily_series"
	if err := s.ensureCollection(ctx, collectionName); err != nil { return nil, err }
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{ConditionOneOf: &qdrant.Condition_Field{Field: &qdrant.FieldCondition{Key: "type", Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: "sales_daily"}}}}},
			{ConditionOneOf: &qdrant.Condition_Field{Field: &qdrant.FieldCondition{Key: "product_id", Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: productID}}}}},
		},
	}
	limit := uint32(100000)
	withPayload := true
	sctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	res, err := s.qdrantClient.Scroll(sctx, &qdrant.ScrollPoints{
		CollectionName: collectionName,
		Filter:         filter,
		Limit:          &limit,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: withPayload}},
	})
	if err != nil { return nil, fmt.Errorf("failed to scroll sales series: %w", err) }
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")
	out := make([]struct{ Date string; Sales float64 }, 0, len(res.GetResult()))
	for _, p := range res.GetResult() {
		if p.Payload == nil { continue }
		d := getStringFromPayload(p.Payload, "date")
		if d == "" || d < startStr || d > endStr { continue }
		var v float64
		if pv := p.Payload["sales"]; pv != nil { v = pv.GetDoubleValue() }
		out = append(out, struct{ Date string; Sales float64 }{Date: d, Sales: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out, nil
}

// RecreateCollection コレクションを削除して再作成（全データ削除）
func (s *VectorStoreService) RecreateCollection(ctx context.Context, collectionName string) error {
	// コレクションを削除
	deleteCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := s.qdrantCollectionsClient.Delete(deleteCtx, &qdrant.DeleteCollection{
		CollectionName: collectionName,
	})

	if err != nil {
		log.Printf("警告: コレクション削除に失敗（続行します）: %v", err)
	} else {
		log.Printf("コレクション '%s' を削除しました", collectionName)
	}

	// コレクションを再作成
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
		return fmt.Errorf("コレクション再作成に失敗: %w", err)
	}

	log.Printf("コレクション '%s' を再作成しました", collectionName)
	return nil
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

		// file_name フィールドにインデックスを作成（フィルタ検索用）
		indexCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		fieldType := qdrant.FieldType_FieldTypeKeyword
		_, err = s.qdrantClient.CreateFieldIndex(indexCtx, &qdrant.CreateFieldIndexCollection{
			CollectionName: collectionName,
			FieldName:      "file_name",
			FieldType:      &fieldType,
		})
		if err != nil {
			log.Printf("⚠️ file_name インデックス作成に失敗（続行します）: %v", err)
		} else {
			log.Printf("✅ file_name フィールドにインデックスを作成しました")
		}

		log.Printf("📌 重要: 'type' フィールドでフィルタリングするには、Qdrantに自動インデックスが作成されます")
	} else {
		log.Printf("コレクション '%s' は既に存在します", collectionName)

		// 既存コレクションにもインデックスが必要か確認して作成
		indexCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		fieldType := qdrant.FieldType_FieldTypeKeyword
		_, err := s.qdrantClient.CreateFieldIndex(indexCtx, &qdrant.CreateFieldIndexCollection{
			CollectionName: collectionName,
			FieldName:      "file_name",
			FieldType:      &fieldType,
		})
		if err != nil {
			// インデックスが既に存在する場合はエラーになるが、問題ない
			log.Printf("  💡 file_name インデックスは既に存在するか、作成不要です")
		} else {
			log.Printf("  ✅ 既存コレクションに file_name インデックスを追加しました")
		}
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

// HasAnomalyResponse は指定された異常に対する回答が既に存在するかを確認します
func (s *VectorStoreService) HasAnomalyResponse(ctx context.Context, anomalyDate string, productID string) (bool, error) {
	collectionName := "anomaly_responses"

	// anomaly_date と product_id でフィルタリング
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "anomaly_date",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: anomalyDate,
							},
						},
					},
				},
			},
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "product_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: productID,
							},
						},
					},
				},
			},
		},
	}

	// 検索を実行（テキストはダミーでOK、フィルタが主目的）
	// topK=1で十分
	searchResults, err := s.SearchWithFilter(ctx, collectionName, "anomaly check", 1, filter)
	if err != nil {
		return false, fmt.Errorf("異常回答の存在確認中に検索エラー: %w", err)
	}

	// 結果が1件以上あれば、回答は存在すると判断
	return len(searchResults) > 0, nil
}

// ========================================
// 深掘り質問機能のための新しい関数
// ========================================

// SaveAnomalyResponseSession 異常回答セッション全体をQdrantに保存
func (s *VectorStoreService) SaveAnomalyResponseSession(ctx context.Context, session *models.AnomalyResponseSession) error {
	collectionName := "anomaly_response_sessions"

	// セッション全体をJSONに変換
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("セッションのJSON化に失敗: %w", err)
	}

	// 検索用のテキストを構築（全会話を連結）
	var conversationText string
	for i, conv := range session.Conversations {
		conversationText += fmt.Sprintf("\n質問%d: %s\n回答%d: %s", i+1, conv.Question, i+1, conv.Answer)
	}

	searchText := fmt.Sprintf(
		"日付: %s\n製品ID: %s\n会話履歴:%s\nタグ: %s\n影響: %s",
		session.AnomalyDate,
		session.ProductID,
		conversationText,
		session.FinalTags,
		session.FinalImpact,
	)

	// メタデータを準備
	metadata := map[string]interface{}{
		"type":               "anomaly_response_session",
		"session_id":         session.SessionID,
		"anomaly_date":       session.AnomalyDate,
		"product_id":         session.ProductID,
		"is_complete":        session.IsComplete,
		"follow_up_count":    session.FollowUpCount,
		"conversation_count": len(session.Conversations),
		"created_at":         session.CreatedAt,
		"session_json":       string(sessionJSON), // 完全なセッションデータを保存
	}

	if session.CompletedAt != "" {
		metadata["completed_at"] = session.CompletedAt
	}

	// Qdrantに保存
	err = s.StoreDocument(ctx, collectionName, session.SessionID, searchText, metadata)
	if err != nil {
		return fmt.Errorf("セッションの保存に失敗: %w", err)
	}

	log.Printf("✅ セッションを保存しました: %s (完了: %v, 会話数: %d)",
		session.SessionID, session.IsComplete, len(session.Conversations))

	return nil
}

// GetAnomalyResponseSession セッションIDから異常回答セッションを取得
func (s *VectorStoreService) GetAnomalyResponseSession(ctx context.Context, sessionID string) (*models.AnomalyResponseSession, error) {
	collectionName := "anomaly_response_sessions"

	// Qdrantから取得
	points, err := s.qdrantClient.Get(ctx, &qdrant.GetPoints{
		CollectionName: collectionName,
		Ids:            []*qdrant.PointId{{PointIdOptions: &qdrant.PointId_Uuid{Uuid: sessionID}}},
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})

	if err != nil {
		return nil, fmt.Errorf("セッション取得に失敗: %w", err)
	}

	if len(points.GetResult()) == 0 {
		return nil, fmt.Errorf("セッションが見つかりません: %s", sessionID)
	}

	// session_jsonフィールドから完全なセッションデータを復元
	payload := points.GetResult()[0].Payload
	sessionJSONValue, ok := payload["session_json"]
	if !ok {
		return nil, fmt.Errorf("session_jsonフィールドが見つかりません")
	}

	sessionJSONStr := ""
	if strVal := sessionJSONValue.GetStringValue(); strVal != "" {
		sessionJSONStr = strVal
	} else {
		return nil, fmt.Errorf("session_jsonの値が取得できません")
	}

	// JSONをパース
	var session models.AnomalyResponseSession
	if err := json.Unmarshal([]byte(sessionJSONStr), &session); err != nil {
		return nil, fmt.Errorf("セッションのJSON解析に失敗: %w", err)
	}

	return &session, nil
}
