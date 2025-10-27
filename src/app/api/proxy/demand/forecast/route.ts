import { proxyRequest } from '@/lib/proxy-helper';

export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
  const requestBody = await request.json();

  return proxyRequest('/api/v1/demand/forecast', {
    method: 'POST',
    body: requestBody,
  });
}