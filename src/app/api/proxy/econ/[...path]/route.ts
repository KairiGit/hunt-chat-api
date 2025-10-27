import { proxyRequest } from '@/lib/proxy-helper';
import { NextRequest } from 'next/server';

export const dynamic = 'force-dynamic';

async function proxy(method: string, request: NextRequest, segments: string[]) {
  const url = new URL(request.url);
  const rest = segments.join('/');
  const endpoint = `/api/v1/econ/${rest}`;

  const contentType = request.headers.get('content-type') || '';
  let body: any;

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

export async function GET(request: NextRequest, context: { params: { path: string[] } }) {
  return proxy('GET', request, context.params.path || []);
}

export async function POST(request: NextRequest, context: { params: { path: string[] } }) {
  return proxy('POST', request, context.params.path || []);
}

export async function PUT(request: NextRequest, context: { params: { path: string[] } }) {
  return proxy('PUT', request, context.params.path || []);
}

export async function DELETE(request: NextRequest, context: { params: { path: string[] } }) {
  return proxy('DELETE', request, context.params.path || []);
}

export async function PATCH(request: NextRequest, context: { params: { path: string[] } }) {
  return proxy('PATCH', request, context.params.path || []);
}
