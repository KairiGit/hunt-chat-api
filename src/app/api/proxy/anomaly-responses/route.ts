import { proxyRequest } from '@/lib/proxy-helper';

export const dynamic = 'force-dynamic';

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  
  return proxyRequest('/api/v1/ai/anomaly-responses', {
    method: 'GET',
    searchParams,
  });
}

export async function DELETE() {
  return proxyRequest('/api/v1/ai/anomaly-responses', {
    method: 'DELETE',
  });
}
