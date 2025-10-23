# Vercelデバッグガイド 🔍

## 📅 作成日
2025年10月15日

## 🎯 目的

ローカル環境では動作するが、Vercel環境でエラーが発生する場合のデバッグ方法をまとめます。

## 🐛 よくある問題と解決策

### 1. JSONパースエラー

#### エラー例
```
SyntaxError: Unexpected non-whitespace character after JSON at position 4
```

#### 原因
- バックエンドからのレスポンスがJSON形式ではない
- HTMLエラーページが返されている
- 404ページが返されている

#### 解決策
✅ **実装済み**: `proxy-helper.ts`でレスポンスの型をチェック

```typescript
// レスポンステキストを取得（デバッグ用）
const responseText = await response.text();
console.log('[Proxy] Response text:', responseText.substring(0, 500));

// JSONパース試行
try {
  data = JSON.parse(responseText);
} catch (parseError) {
  // 詳細なエラー情報を返す
  return new Response(JSON.stringify({
    error: 'Backend returned non-JSON response',
    responseText: responseText.substring(0, 1000),
  }), { status: 502 });
}
```

### 2. 404エラー

#### エラー例
```
POST /api/v1/ai/detect-anomalies → 404
```

#### 原因
- エンドポイントが直接呼ばれている（プロキシ経由ではない）
- Vercel環境ではバックエンドのルーティングが異なる

#### 解決策
✅ **実装済み**: すべてのAPIコールをプロキシ経由に変更

**変更前:**
```typescript
fetch('/api/v1/ai/detect-anomalies', { ... })
```

**変更後:**
```typescript
fetch('/api/proxy/detect-anomalies', { ... })
```

### 3. 環境変数の問題

#### 症状
- ローカルでは動くがVercelでは動かない
- API_KEYエラー
- バックエンドURL not found

#### 確認方法

**Vercelログで確認:**
```
[Proxy] Environment: {
  VERCEL_URL: 'xxx.vercel.app',
  GO_BACKEND_URL: 'not set'
}
```

#### 解決策

1. **Vercelの環境変数を設定**
   - Vercel Dashboard → Settings → Environment Variables
   - `GO_BACKEND_URL`: バックエンドのURL
   - `API_KEY`: APIキー

2. **環境変数の優先順位**
   ```typescript
   // 1. ローカル開発環境
   if (process.env.GO_BACKEND_URL) {
     return process.env.GO_BACKEND_URL;
   }
   
   // 2. Vercel環境
   if (process.env.VERCEL_URL) {
     return `https://${process.env.VERCEL_URL}`;
   }
   
   // 3. フォールバック
   return 'http://localhost:8080';
   ```

## 🔍 Vercelでのログ確認方法

### 1. Vercelダッシュボード

1. プロジェクトページを開く
2. **Deployments** タブをクリック
3. 最新のデプロイメントをクリック
4. **Functions** タブでログを確認

### 2. リアルタイムログ

```bash
# Vercel CLIをインストール
npm i -g vercel

# ログをストリーミング
vercel logs --follow
```

### 3. 特定の関数のログ

```bash
# 特定のAPIルートのログ
vercel logs /api/proxy/learning-insights
```

## 📊 デバッグログの見方

### 正常なログ例

```
[Proxy LOCAL] GET http://localhost:8080/api/v1/ai/learning-insights
[Proxy] Environment: {
  VERCEL_URL: 'not set',
  GO_BACKEND_URL: 'http://localhost:8080'
}
[Proxy] Request options: { method: 'GET', hasBody: false }
[Proxy] Response status: 200
[Proxy] Content-Type: application/json
[Proxy] Response text: {"success":true,"insights":[...]}
```

### エラーログ例

```
[Proxy VERCEL] GET https://xxx.vercel.app/api/v1/ai/learning-insights
[Proxy] Environment: {
  VERCEL_URL: 'xxx.vercel.app',
  GO_BACKEND_URL: 'not set'
}
[Proxy] Response status: 404
[Proxy] Content-Type: text/html
[Proxy] Response text: <!DOCTYPE html><html>...
[Proxy] JSON parse error: Unexpected token < in JSON
```

## 🛠️ トラブルシューティングフロー

### ステップ1: ログを確認

1. Vercelログで `[Proxy]` を検索
2. 環境変数が正しく設定されているか確認
3. リクエストURLを確認
4. レスポンスステータスを確認
5. Content-Typeを確認

### ステップ2: 環境の違いを特定

| 項目 | ローカル | Vercel |
|------|---------|--------|
| バックエンドURL | `GO_BACKEND_URL` | `VERCEL_URL` |
| APIキー | `.env.local` | Vercel設定 |
| ログ出力 | ターミナル | Vercel Dashboard |

### ステップ3: レスポンスの内容を確認

```typescript
// proxy-helper.tsで自動的に出力
console.log('[Proxy] Response text (first 500 chars):', responseText.substring(0, 500));
```

これにより、HTMLエラーページやその他の非JSON レスポンスを特定できます。

### ステップ4: エラーレスポンスの詳細を確認

フロントエンドでエラーレスポンスを表示:

```typescript
try {
  const response = await fetch('/api/proxy/xxx');
  if (!response.ok) {
    const errorData = await response.json();
    console.error('Proxy error details:', errorData);
    // errorData には以下が含まれる:
    // - error: エラーメッセージ
    // - details: 詳細情報
    // - targetUrl: バックエンドのURL
    // - statusCode: HTTPステータスコード
    // - responseText: レスポンス本文（最初の1000文字）
    // - timestamp: エラー発生時刻
    // - environment: 'vercel' or 'local'
  }
} catch (error) {
  console.error('Fetch error:', error);
}
```

## 🎯 新しいAPIエンドポイントを追加する場合

### 1. バックエンドでエンドポイントを実装
```go
// cmd/server/main.go
ai.POST("/new-endpoint", handler.NewEndpoint)
```

### 2. プロキシルートを作成
```typescript
// src/app/api/proxy/new-endpoint/route.ts
import { proxyRequest } from '@/lib/proxy-helper';

export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
  const body = await request.json();
  
  return proxyRequest('/api/v1/ai/new-endpoint', {
    method: 'POST',
    body,
  });
}
```

### 3. フロントエンドで呼び出し
```typescript
const response = await fetch('/api/proxy/new-endpoint', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ ... })
});
```

## 📝 チェックリスト

デプロイ前に確認:

- [ ] すべてのAPIコールがプロキシ経由か？
- [ ] Vercelの環境変数が設定されているか？
  - [ ] `GO_BACKEND_URL` (オプション)
  - [ ] `API_KEY`
- [ ] プロキシルートが作成されているか？
- [ ] `proxy-helper.ts` を使用しているか？
- [ ] ローカルでテストしたか？

デプロイ後に確認:

- [ ] Vercelログでエラーがないか？
- [ ] `[Proxy]` ログが出力されているか？
- [ ] 環境変数が正しく読み込まれているか？
- [ ] レスポンスのContent-Typeが正しいか？
- [ ] すべての機能が動作するか？

## 🚀 パフォーマンス最適化

### 1. ログの削減（本番環境）

```typescript
const isProduction = process.env.NODE_ENV === 'production';

if (!isProduction) {
  console.log('[Proxy] Detailed logs...');
}
```

### 2. タイムアウト設定

```typescript
const controller = new AbortController();
const timeoutId = setTimeout(() => controller.abort(), 30000); // 30秒

try {
  const response = await fetch(targetUrl, {
    signal: controller.signal,
    // ...
  });
} finally {
  clearTimeout(timeoutId);
}
```

### 3. リトライロジック

```typescript
async function fetchWithRetry(url: string, options: any, retries = 3) {
  for (let i = 0; i < retries; i++) {
    try {
      return await fetch(url, options);
    } catch (error) {
      if (i === retries - 1) throw error;
      await new Promise(resolve => setTimeout(resolve, 1000 * (i + 1)));
    }
  }
}
```

## 📚 関連ドキュメント

- `src/lib/proxy-helper.ts`: プロキシヘルパー実装
- `FILE_ANALYSIS_ERROR_FIX.md`: エラーハンドリング
- `WARNING_MESSAGE_IMPROVEMENT.md`: ユーザー向けエラー表示

## 🔗 参考リンク

- [Vercel Functions](https://vercel.com/docs/functions)
- [Vercel Environment Variables](https://vercel.com/docs/projects/environment-variables)
- [Vercel Logs](https://vercel.com/docs/observability/runtime-logs)
