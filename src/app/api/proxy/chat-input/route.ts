import { NextResponse } from 'next/server';

// ストリーミングを有効にするための設定
export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
  try {
    // フロントエンドからのリクエストボディ(JSON)を取得
    const requestBody = await request.json();

    // GoバックエンドのURLを決定
    const baseUrl = process.env.GO_BACKEND_URL || '';
    const targetUrl = `${baseUrl}/api/v1/ai/chat-input`;

    // ヘッダーにAPIキーとContent-Typeを追加
    const headers = new Headers();
    headers.append('Content-Type', 'application/json');
    headers.append('X-API-KEY', process.env.API_KEY || '');

    // Goバックエンドにリクエストを転送
    const response = await fetch(targetUrl, {
      method: 'POST',
      headers: headers,
      body: JSON.stringify(requestBody),
    });

    // Goバックエンドからのストリーミングレスポンスをそのままフロントエンドに中継
    if (response.body) {
      return new NextResponse(response.body, {
        headers: {
          'Content-Type': 'text/plain; charset=utf-8',
        },
      });
    }

    // ボディがない場合（エラーなど）
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
