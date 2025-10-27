import { proxyRequest } from '@/lib/proxy-helper';
import { NextResponse } from 'next/server';

export const dynamic = 'force-dynamic';

export async function GET() {
  return NextResponse.json({ ok: true, route: '/api/proxy/econ/import' });
}

export async function POST(request: Request) {
  const contentType = request.headers.get('content-type') || '';
  let body: string | FormData | Record<string, unknown>;

  if (contentType.includes('application/json')) {
    body = await request.json();
  } else if (contentType.includes('multipart/form-data')) {
    body = await request.formData();
  } else {
    body = await request.text();
  }

  return proxyRequest('/api/v1/econ/import', {
    method: 'POST',
    body: body,
  });
}
