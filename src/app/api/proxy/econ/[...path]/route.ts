import { NextRequest, NextResponse } from 'next/server';

export const dynamic = 'force-dynamic';

function buildBaseUrl() {
  const isDev = process.env.NODE_ENV !== 'production';
  return (
    process.env.GO_BACKEND_URL?.trim() ||
    process.env.NEXT_PUBLIC_BACKEND_URL?.trim() ||
    (isDev
      ? 'http://localhost:8080'
      : process.env.VERCEL_URL
        ? `https://${process.env.VERCEL_URL}`
        : 'https://hunt-chat-api.vercel.app')
  );
}

async function proxy(method: 'GET'|'POST'|'PUT'|'DELETE'|'PATCH', request: NextRequest, segments: string[]) {
  let targetUrl = '';
  try {
    const baseUrl = buildBaseUrl();
    const url = new URL(request.url);
    const rest = segments.join('/');
    const qs = url.search ? url.search : '';
    targetUrl = `${baseUrl}/api/v1/econ/${rest}${qs}`;

    const headers = new Headers();
    headers.append('X-API-KEY', process.env.API_KEY || '');

    let body: BodyInit | undefined;
    if (method !== 'GET') {
      const contentType = request.headers.get('content-type') || '';
      if (contentType.includes('application/json')) {
        const json = await request.json();
        headers.append('Content-Type', 'application/json');
        body = JSON.stringify(json);
      } else if (contentType.includes('multipart/form-data')) {
        const form = await request.formData();
        const f = new FormData();
        for (const [key, value] of form.entries()) {
          if (typeof value === 'string') {
            f.append(key, value);
          } else {
            f.append(key, value, (value as File).name);
          }
        }
        body = f;
      } else if (contentType) {
        // passthrough text or other
        const text = await request.text();
        headers.append('Content-Type', contentType);
        body = text;
      } else {
        // no content-type -> try raw text
        const text = await request.text();
        if (text) {
          headers.append('Content-Type', 'text/plain; charset=utf-8');
          body = text;
        }
      }
    }

    const resp = await fetch(targetUrl, { method, headers, body });
    const ct = resp.headers.get('content-type') || '';
    if (ct.includes('application/json')) {
      const data = await resp.json();
      return NextResponse.json(data, { status: resp.status });
    }
    const text = await resp.text().catch(() => '');
    return NextResponse.json({ error: text || 'non-json response' }, { status: resp.status });
  } catch (error) {
    console.error('Proxy error in /api/proxy/econ/[...path]:', error);
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    return NextResponse.json(
      { error: 'Proxy error', details: `Failed to reach ${targetUrl}`, proxyError: errorMessage },
      { status: 500 }
    );
  }
}

export async function GET(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  return proxy('GET', request, path || []);
}

export async function POST(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  return proxy('POST', request, path || []);
}

export async function PUT(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  return proxy('PUT', request, path || []);
}

export async function DELETE(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  return proxy('DELETE', request, path || []);
}

export async function PATCH(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  return proxy('PATCH', request, path || []);
}
