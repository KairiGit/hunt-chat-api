import { proxyRequest } from '@/lib/proxy-helper';

export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
  const body = await request.json();
  
  return proxyRequest('/api/v1/ai/detect-anomalies', {
    method: 'POST',
    body,
  });
}
