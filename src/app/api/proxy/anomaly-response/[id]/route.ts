import { NextRequest, NextResponse } from 'next/server';

const GO_BACKEND_URL = process.env.GO_BACKEND_URL || 'https://hunt-chat-api.vercel.app';

export async function DELETE(
  request: NextRequest,
  { params }: { params: { id: string } }
) {
  try {
    const { id } = params;
    
    const apiUrl = `${GO_BACKEND_URL}/api/v1/ai/anomaly-response/${id}`;
    
    console.log(`[Proxy DELETE /anomaly-response/${id}] Forwarding to: ${apiUrl}`);
    
    const response = await fetch(apiUrl, {
      method: 'DELETE',
      headers: {
        'X-API-KEY': process.env.API_KEY || '',
      },
    });

    // レスポンスがJSONでない場合の処理
    const contentType = response.headers.get('content-type');
    if (!contentType || !contentType.includes('application/json')) {
      const text = await response.text();
      console.error('[Proxy DELETE /anomaly-response/:id] Non-JSON response:', text);
      return NextResponse.json(
        { success: false, error: `サーバーエラー: ${text}` },
        { status: response.status }
      );
    }

    const data = await response.json();
    
    return NextResponse.json(data, { status: response.status });
  } catch (error) {
    console.error('[Proxy DELETE /anomaly-response/:id] Error:', error);
    return NextResponse.json(
      { success: false, error: 'プロキシエラーが発生しました' },
      { status: 500 }
    );
  }
}
