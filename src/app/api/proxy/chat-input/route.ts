import { NextResponse } from 'next/server';

export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
  let targetUrl = '';
  try {
    const requestBody = await request.json();

    const baseUrl = process.env.GO_BACKEND_URL || `https://${process.env.VERCEL_URL}`;
    targetUrl = `${baseUrl}/api/v1/ai/chat-input`;

    const headers = new Headers();
    headers.append('Content-Type', 'application/json');
    headers.append('X-API-KEY', process.env.API_KEY || '');

    const response = await fetch(targetUrl, {
      method: 'POST',
      headers: headers,
      body: JSON.stringify(requestBody),
    });

    if (response.body) {
      return new NextResponse(response.body, {
        headers: {
          'Content-Type': 'text/plain; charset=utf-8',
        },
      });
    }

    return new NextResponse(response.body, {
      status: response.status,
      statusText: response.statusText,
      headers: response.headers,
    });

  } catch (error) {
    console.error('Proxy error in /api/proxy/chat-input:', error);
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