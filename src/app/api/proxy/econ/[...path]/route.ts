import { proxyRequest } from '@/lib/proxy-helper';
import { NextRequest } from 'next/server';

export const dynamic = 'force-dynamic';

async function proxy(method: string, request: NextRequest, segments: string[]) {
  const url = new URL(request.url);
  const rest = segments.join('/');
  const endpoint = `/api/v1/econ/${rest}`;

  const contentType = request.headers.get('content-type') || '';
  let body: string | FormData | Record<string, unknown> | undefined;

  if (method !== 'GET') {
    if (contentType.includes('application/json')) {
      body = await request.json();
    } else if (contentType.includes('multipart/form-data')) {
      body = await request.formData();
    } else {
      body = await request.text();
    }
  }

  return proxyRequest(endpoint, {
    method,
    body,
    searchParams: url.searchParams,
  });
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
