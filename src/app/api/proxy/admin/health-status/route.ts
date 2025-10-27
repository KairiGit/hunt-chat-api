import { proxyRequest } from '@/lib/proxy-helper';

export const dynamic = 'force-dynamic';

export async function GET() {
  return proxyRequest('/admin/health-status', {
    method: 'GET',
  });
}
