import { proxyRequest } from '@/lib/proxy-helper';
import { NextRequest } from 'next/server';

export const dynamic = 'force-dynamic';

export async function POST(request: NextRequest) {
  const body = await request.json();
  return proxyRequest('/api/v1/ai/anomaly-response', {
    method: 'POST',
    body,
  });
}