import { NextResponse } from 'next/server';

export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
  let targetUrl = '';
  try {
    const requestBody = await request.json();

    const baseUrl = process.env.GO_BACKEND_URL || `https://${process.env.VERCEL_URL}`;
    targetUrl = `${baseUrl}/api/v1/demand/forecast`;

    const headers = new Headers();
    headers.append('Content-Type', 'application/json');
    headers.append('X-API-KEY', process.env.API_KEY || '');

    const response = await fetch(targetUrl, {
      method: 'POST',
      headers: headers,
      body: JSON.stringify(requestBody),
    });

    const data = await response.json();

    return NextResponse.json(data, {
      status: response.status,
    });

  } catch (error) {
    console.error('Proxy error in /api/proxy/demand/forecast:', error);
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    return NextResponse.json(
      {
        error: 'An internal server error occurred in the proxy.',
        details: `Failed to fetch target: ${targetUrl}`,
        proxyError: errorMessage,
      },
      { status: 500 }
    );
  }
}