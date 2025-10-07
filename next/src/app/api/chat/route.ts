import { NextRequest, NextResponse } from 'next/server';

// Go APIのベースURL
const GO_API_BASE_URL = 'http://localhost:8080/api/v1';

/**
 * POST: チャット入力とファイルをGoバックエンドに転送
 */
export async function POST(request: NextRequest) {
  try {
    // フロントエンドから送られてきたFormDataを取得
    const formData = await request.formData();

    // Goバックエンドに同じFormDataを送信
    const response = await fetch(`${GO_API_BASE_URL}/ai/chat-input`, {
      method: 'POST',
      body: formData,
      // Content-Typeはfetchが自動で設定するため、ここでは指定しない
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Failed to parse error response' }));
      console.error('Go API Error:', response.status, errorData);
      return NextResponse.json({ error: 'Failed to forward request to Go API', details: errorData }, { status: response.status });
    }

    const data = await response.json();
    return NextResponse.json(data);

  } catch (error) {
    console.error('Internal Server Error:', error);
    return NextResponse.json({ error: 'Internal Server Error' }, { status: 500 });
  }
}
