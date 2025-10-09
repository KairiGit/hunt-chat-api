import { NextResponse } from 'next/server';

// Go APIのベースURL
const GO_API_BASE_URL = 'http://localhost:8080/api/v1';

/**
 * GET: 需要予測の設定情報を取得
 * Goバックエンドの /api/v1/demand/settings エンドポイントから設定を取得します。
 */
export async function GET() {
  try {
    const response = await fetch(`${GO_API_BASE_URL}/demand/settings`, {
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Failed to parse error response' }));
      console.error('Go API Error:', response.status, errorData);
      return NextResponse.json({ error: 'Failed to fetch settings from Go API', details: errorData }, { status: response.status });
    }

    const data = await response.json();
    return NextResponse.json(data);

  } catch (error) {
    console.error('Internal Server Error:', error);
    return NextResponse.json({ error: 'Internal Server Error' }, { status: 500 });
  }
}

/**
 * POST: 需要予測を実行
 * フロントエンドから受け取ったリクエストをGoバックエンドの /api/v1/demand/forecast に転送します。
 */
export async function POST(request: Request) {
  try {
    const body = await request.json();

    // デフォルト値の補完
    const requestBody = {
      region_code: body.region_code || '240000',
      product_category: body.product_category || '飲料',
      forecast_days: body.forecast_days || 7,
      historical_days: 30, // 固定または将来的にUIから設定
      // 以下はGoバックエンドのデフォルトに任せるか、必要に応じてUIから受け取る
      tacit_knowledge: [],
      seasonal_factors: {},
      external_factors: {},
      ...body, // UIから来た値で上書き
    };

    const response = await fetch(`${GO_API_BASE_URL}/demand/forecast`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(requestBody),
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Failed to parse error response' }));
      console.error('Go API Error:', response.status, errorData);
      return NextResponse.json({ error: 'Failed to fetch forecast from Go API', details: errorData }, { status: response.status });
    }

    const data = await response.json();
    return NextResponse.json(data);

  } catch (error) {
    console.error('Internal Server Error:', error);
    return NextResponse.json({ error: 'Internal Server Error' }, { status: 500 });
  }
}
