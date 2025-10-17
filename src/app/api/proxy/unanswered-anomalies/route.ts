import { proxyRequest } from '@/lib/proxy-helper';
import { NextRequest } from 'next/server';

export const dynamic = 'force-dynamic';

export async function GET(request: NextRequest) {
  return proxyRequest('/api/v1/ai/unanswered-anomalies', {
    method: 'GET',
  });
}
