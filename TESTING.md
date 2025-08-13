# HUNT Chat-API テストドキュメント

## 📋 概要

HUNT Chat-APIプロジェクトには、アプリケーションの品質と信頼性を確保するための包括的なテストスイートが含まれています。このドキュメントでは、実装されているテストの詳細と実行方法について説明します。

## 🧪 テスト構成

### テストファイル一覧

| ファイル | 場所 | 目的 |
|---------|------|------|
| `config_test.go` | `configs/` | 設定管理機能のテスト |
| `weather_service_test.go` | `internal/services/` | 気象データサービスのテスト |
| `handlers_test.go` | `internal/handlers/` | HTTPハンドラーのテスト |
| `server_test.go` | `cmd/server/` | アプリケーション統合テスト |

## 🔧 設定テスト (`configs/config_test.go`)

### テスト対象
- 環境変数からの設定読み込み
- デフォルト値の適用
- 設定値の検証

### テストケース

#### `TestLoadConfig`
```go
func TestLoadConfig(t *testing.T)
```
**目的**: 環境変数から正しく設定が読み込まれることを確認

**テスト内容**:
- Azure OpenAI関連の環境変数設定
- ポート番号とエンvironment設定
- 設定値の正確性確認

**検証項目**:
- `PORT`: "8080"
- `ENVIRONMENT`: "test"
- `AZURE_OPENAI_ENDPOINT`: "https://test.openai.azure.com/"
- `AZURE_OPENAI_API_KEY`: "test-key"
- `AZURE_OPENAI_MODEL`: "gpt-4"

#### `TestLoadConfigDefaults`
```go
func TestLoadConfigDefaults(t *testing.T)
```
**目的**: 環境変数が設定されていない場合のデフォルト値確認

**検証項目**:
- デフォルトポート: "8080"
- デフォルト環境: "development"

## 🌤️ WeatherServiceテスト (`internal/services/weather_service_test.go`)

### テスト対象
- WeatherServiceの初期化
- 地域コード管理
- データ範囲取得機能

### テストケース

#### `TestNewWeatherService`
```go
func TestNewWeatherService(t *testing.T)
```
**目的**: WeatherServiceが正しく初期化されることを確認

**検証項目**:
- サービスインスタンスがnilでない
- HTTPクライアントが正しく設定されている

#### `TestGetRegionCodes`
```go
func TestGetRegionCodes(t *testing.T)
```
**目的**: 地域コード取得機能の動作確認

**検証項目**:
- 地域コードマップが空でない
- 東京都コード（130000）の存在確認
- 三重県コード（240000）の存在確認

#### `TestGetRegionName`
```go
func TestGetRegionName(t *testing.T)
```
**目的**: 地域コードから地域名への変換機能確認

**テストケース**:
| 地域コード | 期待される地域名 |
|------------|------------------|
| 130000 | 東京都 |
| 240000 | 三重県 |
| 999999 | 不明な地域 |

#### `TestGetAvailableHistoricalDataRange`
```go
func TestGetAvailableHistoricalDataRange(t *testing.T)
```
**目的**: 利用可能な過去データ範囲の取得機能確認

**検証項目**:
- `start_date`: 開始日
- `end_date`: 終了日
- `max_range_days`: 最大取得日数
- `data_sources`: データソース情報

## 🌐 ハンドラーテスト (`internal/handlers/handlers_test.go`)

### テスト対象
- HTTPエンドポイントの動作
- JSONレスポンスの形式
- ハンドラーの初期化

### テストケース

#### `TestHealthCheck`
```go
func TestHealthCheck(t *testing.T)
```
**目的**: ヘルスチェックエンドポイントの動作確認

**エンドポイント**: `GET /health`

**検証項目**:
- HTTPステータス: 200 OK
- レスポンスに`status`フィールドが含まれる
- レスポンスに`service`フィールドが含まれる

#### `TestHelloAPI`
```go
func TestHelloAPI(t *testing.T)
```
**目的**: Hello APIエンドポイントの動作確認

**エンドポイント**: `GET /api/v1/hello`

**検証項目**:
- HTTPステータス: 200 OK
- レスポンスに期待されるメッセージが含まれる

#### `TestWeatherHandlerCreation`
```go
func TestWeatherHandlerCreation(t *testing.T)
```
**目的**: WeatherHandlerの正しい初期化確認

**検証項目**:
- ハンドラーインスタンスがnilでない
- WeatherServiceが正しく設定されている

#### `TestWeatherRegionCodesEndpoint`
```go
func TestWeatherRegionCodesEndpoint(t *testing.T)
```
**目的**: 気象データ地域コードAPIの動作確認

**エンドポイント**: `GET /api/v1/weather/regions`

**検証項目**:
- HTTPステータス: 200 OK
- レスポンスに`success`フィールドが含まれる
- レスポンスに`data`フィールドが含まれる

## 🏗️ 統合テスト (`cmd/server/server_test.go`)

### テスト対象
- アプリケーション全体の初期化
- サービス間の連携
- ルーター設定
- 環境変数の管理

### テストケース

#### `TestApplicationSetup`
```go
func TestApplicationSetup(t *testing.T)
```
**目的**: アプリケーション全体の初期化プロセス確認

**検証項目**:
- Config読み込み
- AzureOpenAIService初期化
- WeatherHandler初期化
- DemandForecastHandler初期化
- AIHandler初期化

#### `TestRouterSetup`
```go
func TestRouterSetup(t *testing.T)
```
**目的**: Ginルーターの正しい設定確認

**検証エンドポイント**:
- `GET /health`
- `GET /api/v1/hello`

#### `TestEnvironmentVariables`
```go
func TestEnvironmentVariables(t *testing.T)
```
**目的**: 重要な環境変数の設定確認

**検証変数**:
- `AZURE_OPENAI_ENDPOINT`
- `AZURE_OPENAI_API_KEY`
- `AZURE_OPENAI_MODEL`

## 🚀 テスト実行方法

### 全テスト実行
```bash
make test
```

### 特定パッケージのテスト実行
```bash
# 設定テスト
go test -v ./configs

# サービステスト
go test -v ./internal/services

# ハンドラーテスト
go test -v ./internal/handlers

# 統合テスト
go test -v ./cmd/server
```

### テスト結果例
```
=== RUN   TestApplicationSetup
--- PASS: TestApplicationSetup (0.00s)
=== RUN   TestRouterSetup
--- PASS: TestRouterSetup (0.00s)
=== RUN   TestEnvironmentVariables
--- PASS: TestEnvironmentVariables (0.00s)
PASS
ok      hunt-chat-api/cmd/server        0.023s
```

## 📊 テストカバレッジ

### 現在のテストカバレッジ
- **設定管理**: ✅ 基本機能カバー済み
- **WeatherService**: ✅ 主要機能カバー済み
- **HTTPハンドラー**: ✅ 基本エンドポイントカバー済み
- **統合テスト**: ✅ アプリケーション初期化カバー済み

### 追加推奨テスト
- [ ] Azure OpenAI Service統合テスト
- [ ] 需要予測機能テスト
- [ ] エラーハンドリングテスト
- [ ] APIレスポンス形式テスト
- [ ] パフォーマンステスト

## 🔍 テストのベストプラクティス

### 1. 環境変数管理
```go
// テスト用環境変数の設定と後片付け
defer func() {
    for key := range testCases {
        os.Unsetenv(key)
    }
}()
```

### 2. HTTPテスト
```go
// Ginのテストモードを使用
gin.SetMode(gin.TestMode)

// httptest.NewRecorderを使用してレスポンステスト
w := httptest.NewRecorder()
router.ServeHTTP(w, req)
```

### 3. アサーション
```go
// stretchr/testifyを使用した分かりやすいアサーション
assert.Equal(t, http.StatusOK, w.Code)
assert.NotEmpty(t, value, "Environment variable %s should not be empty", envVar)
```

## 🚧 今後の拡張計画

### 追加予定のテスト

1. **エンドツーエンドテスト**
   - 実際のAPIリクエスト/レスポンステスト
   - データベース統合テスト

2. **パフォーマンステスト**
   - 負荷テスト
   - レスポンス時間測定

3. **セキュリティテスト**
   - 認証・認可テスト
   - 入力値検証テスト

4. **モックテスト**
   - 外部API依存関係のモック化
   - より詳細な単体テスト

## 📝 テスト作成ガイドライン

### 新しいテストを追加する場合

1. **ファイル命名規則**: `*_test.go`
2. **関数命名規則**: `TestFunctionName`
3. **パッケージ**: テスト対象と同じパッケージ
4. **クリーンアップ**: `defer`を使用したリソース解放
5. **アサーション**: `testify/assert`を使用
6. **ドキュメント**: テストの目的と検証項目を明記

### テスト品質チェックリスト

- [ ] テスト名が明確で理解しやすい
- [ ] 各テストが独立して実行可能
- [ ] 適切なエラーメッセージが含まれている
- [ ] テスト後のクリーンアップが実装されている
- [ ] エッジケースも考慮されている

---

このテストスイートにより、HUNT Chat-APIの品質と信頼性が継続的に保たれています。新機能の追加時には、対応するテストも併せて作成することを推奨します。
