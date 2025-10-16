import { proxyRequest } from '@/lib/proxy-helper';

export const dynamic = 'force-dynamic';

export async function GET(request: Request) {
  return proxyRequest('/api/v1/ai/analysis-reports', {
    method: 'GET',
  });
}

export async function DELETE(request: Request) {
  return proxyRequest('/api/v1/ai/analysis-reports', {
    method: 'DELETE',
  });
}
