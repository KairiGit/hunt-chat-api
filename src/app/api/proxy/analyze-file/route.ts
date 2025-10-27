import { proxyRequest } from '@/lib/proxy-helper';

export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
  const formData = await request.formData();

  return proxyRequest('/api/v1/ai/analyze-file', {
    method: 'POST',
    body: formData,
  });
}