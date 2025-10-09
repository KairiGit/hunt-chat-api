import { NextResponse } from 'next/server';

export async function POST(request: Request) {
  try {
    // フロントエンドからのリクエストボディを取得
    const formData = await request.formData();

    // GoバックエンドのURLを決定
    console.log('DEBUG: GO_BACKEND_URL is:', process.env.GO_BACKEND_URL);
    // Vercel環境ではGo APIは同じルートにあるため相対パスでOK
    // ローカル環境では環境変数から取得（例: http://localhost:8080）
    const baseUrl = process.env.GO_BACKEND_URL || '';
    const targetUrl = `${baseUrl}/api/v1/ai/analyze-file`;

    // ヘッダーにAPIキーを追加
    const headers = new Headers();
    headers.append('X-API-KEY', process.env.API_KEY || '');

    // Goバックエンドにリクエストを転送
    const response = await fetch(targetUrl, {
      method: 'POST',
      headers: headers,
      body: formData,
    });

    // Goバックエンドからのレスポンスをそのままフロントエンドに返す
    return new NextResponse(response.body, {
      status: response.status,
      statusText: response.statusText,
      headers: response.headers,
    });

  } catch (error) {
    console.error('Proxy error:', error);
    return NextResponse.json(
      { error: 'An internal server error occurred in the proxy.' },
      { status: 500 }
    );
  }
}
