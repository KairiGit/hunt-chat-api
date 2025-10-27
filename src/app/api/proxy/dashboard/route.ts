import { proxyRequest } from '@/lib/proxy-helper';
import { NextRequest } from 'next/server';

export const dynamic = 'force-dynamic';

export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url);

  return proxyRequest('/monitoring/logs', {
    method: 'GET',
    searchParams: searchParams,
  });
}
