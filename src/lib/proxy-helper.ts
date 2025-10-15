/**
 * Vercelデバッグ対応のプロキシヘルパー
 * - 詳細なエラーログ
 * - レスポンスの型チェック
 * - 環境別のURL設定
 */

export interface ProxyErrorDetails {
  error: string;
  details?: string;
  targetUrl?: string;
  statusCode?: number;
  responseText?: string;
  timestamp?: string;
  environment?: 'vercel' | 'local';
}

/**
 * 環境に応じたバックエンドのベースURLを取得
 */
export function getBackendBaseUrl(): string {
  // 1. 明示的に設定されたGO_BACKEND_URLを最優先
  if (process.env.GO_BACKEND_URL) {
    console.log('[Proxy] Using GO_BACKEND_URL:', process.env.GO_BACKEND_URL);
    return process.env.GO_BACKEND_URL;
  }

  // 2. Vercel環境の場合
  if (process.env.VERCEL_URL) {
    const vercelUrl = `https://${process.env.VERCEL_URL}`;
    console.log('[Proxy] Using VERCEL_URL:', vercelUrl);
    return vercelUrl;
  }

  // 3. ローカル開発環境
  const localUrl = 'http://localhost:8080';
  console.log('[Proxy] Using local URL:', localUrl);
  return localUrl;
}

/**
 * バックエンドAPIへのプロキシリクエストを実行
 * 
 * @param endpoint - バックエンドのエンドポイントパス（例: '/api/v1/ai/chat'）
 * @param options - fetch options（method, body, headersなど）
 * @returns Response object
 */
export async function proxyRequest(
  endpoint: string,
  options: {
    method?: string;
    body?: Record<string, unknown> | Array<unknown>;
    headers?: Record<string, string>;
    searchParams?: URLSearchParams;
  } = {}
): Promise<Response> {
  const startTime = Date.now();
  const isVercel = !!process.env.VERCEL_URL;
  const environment = isVercel ? 'VERCEL' : 'LOCAL';

  // ベースURLを取得
  const baseUrl = getBackendBaseUrl();
  let targetUrl = `${baseUrl}${endpoint}`;
  
  // クエリパラメータを追加
  if (options.searchParams) {
    targetUrl += `?${options.searchParams.toString()}`;
  }

  console.log(`[Proxy ${environment}] ${options.method || 'GET'} ${targetUrl}`);
  console.log('[Proxy] Environment:', {
    VERCEL_URL: process.env.VERCEL_URL || 'not set',
    GO_BACKEND_URL: process.env.GO_BACKEND_URL || 'not set',
  });

  try {
    // ヘッダーの準備
    const headers = new Headers(options.headers || {});
    
    // API Keyを追加
    if (process.env.API_KEY) {
      headers.set('X-API-KEY', process.env.API_KEY);
    }

    // Content-Typeの設定
    if (options.body && !headers.has('Content-Type')) {
      headers.set('Content-Type', 'application/json');
    }

    // リクエストオプション
    const fetchOptions: RequestInit = {
      method: options.method || 'GET',
      headers: headers,
    };

    // ボディがある場合は追加
    if (options.body) {
      fetchOptions.body = JSON.stringify(options.body);
    }

    console.log('[Proxy] Request options:', {
      method: fetchOptions.method,
      hasBody: !!fetchOptions.body,
      headers: Object.fromEntries(headers.entries()),
    });

    // バックエンドへリクエスト
    const response = await fetch(targetUrl, fetchOptions);
    
    const duration = Date.now() - startTime;
    console.log(`[Proxy] Response status: ${response.status} (${duration}ms)`);
    console.log('[Proxy] Content-Type:', response.headers.get('Content-Type'));

    // レスポンステキストを取得（デバッグ用）
    const responseText = await response.text();
    console.log('[Proxy] Response text (first 500 chars):', responseText.substring(0, 500));

    // JSONパース試行
    let data: unknown;
    try {
      data = JSON.parse(responseText);
    } catch (parseError) {
      console.error('[Proxy] JSON parse error:', parseError);
      console.error('[Proxy] Response was not valid JSON');
      
      // 詳細なエラー情報を返す
      const errorDetails: ProxyErrorDetails = {
        error: 'Backend returned non-JSON response',
        details: 'The backend server returned HTML or plain text instead of JSON',
        targetUrl,
        statusCode: response.status,
        responseText: responseText.substring(0, 1000),
        timestamp: new Date().toISOString(),
        environment: isVercel ? 'vercel' : 'local',
      };

      return new Response(JSON.stringify(errorDetails), {
        status: 502,
        headers: {
          'Content-Type': 'application/json',
        },
      });
    }

    // 成功レスポンスを返す
    return new Response(JSON.stringify(data), {
      status: response.status,
      headers: {
        'Content-Type': 'application/json',
      },
    });

  } catch (error) {
    const duration = Date.now() - startTime;
    console.error(`[Proxy] Request failed after ${duration}ms:`, error);

    // エラー詳細を構築
    const errorDetails: ProxyErrorDetails = {
      error: 'Proxy request failed',
      details: error instanceof Error ? error.message : 'Unknown error',
      targetUrl,
      timestamp: new Date().toISOString(),
      environment: isVercel ? 'vercel' : 'local',
    };

    // ネットワークエラーの追加情報
    if (error instanceof TypeError) {
      errorDetails.details = `Network error: ${error.message}. Check if backend is running at ${baseUrl}`;
    }

    console.error('[Proxy] Error details:', errorDetails);

    return new Response(JSON.stringify(errorDetails), {
      status: 500,
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }
}
