import { NextResponse } from 'next/server';

export const dynamic = 'force-dynamic';

export async function GET() {
  return NextResponse.json({ ok: true, route: '/api/proxy/econ/import' });
}

export async function POST(request: Request) {
  let targetUrl = '';
  try {
    const isDev = process.env.NODE_ENV !== 'production';
    const baseUrl =
      process.env.GO_BACKEND_URL?.trim() ||
      process.env.NEXT_PUBLIC_BACKEND_URL?.trim() ||
      (isDev ? 'http://localhost:8080' : (process.env.VERCEL_URL ? `https://${process.env.VERCEL_URL}` : 'https://hunt-chat-api.vercel.app'));
    targetUrl = `${baseUrl}/api/v1/econ/import`;

    const contentType = request.headers.get('content-type') || '';
    const headers = new Headers();
    headers.append('X-API-KEY', process.env.API_KEY || '');

    let body: BodyInit | undefined;

    if (contentType.includes('application/json')) {
      const json = await request.json();
      headers.append('Content-Type', 'application/json');
      body = JSON.stringify(json);
    } else if (contentType.includes('multipart/form-data')) {
      // Recreate FormData to ensure boundary is set correctly by fetch
      const form = await request.formData();
      const f = new FormData();
      for (const [key, value] of form.entries()) {
        if (typeof value === 'string') {
          f.append(key, value);
        } else {
          // File or Blob
          f.append(key, value, (value as File).name);
        }
      }
      body = f;
      // Do NOT set Content-Type manually; fetch will set boundary
    } else {
      // Try raw pass-through as text
      const text = await request.text();
      // Assume CSV text body
      headers.append('Content-Type', 'text/plain; charset=utf-8');
      body = text;
    }

  const response = await fetch(targetUrl, {
      method: 'POST',
      headers,
      body,
    });

  // Prefer JSON response when possible; if not JSON, try to wrap text body
    const ct = response.headers.get('content-type') || '';
    if (ct.includes('application/json')) {
      const data = await response.json();
      return NextResponse.json(data, { status: response.status });
    }

  // Fallback: return text wrapped as JSON for easier frontend handling
  const text = await response.text().catch(() => '');
  return NextResponse.json({ error: text || 'non-json response' }, { status: response.status });

  } catch (error) {
    console.error('Proxy error in /api/proxy/econ/import:', error);
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
