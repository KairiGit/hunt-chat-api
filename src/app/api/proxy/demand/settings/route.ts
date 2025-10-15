import { proxyRequest } from '@/lib/proxy-helper';

export const dynamic = 'force-dynamic';

export async function GET() {
  return proxyRequest('/api/v1/demand/settings', {
    method: 'GET',
  });
}