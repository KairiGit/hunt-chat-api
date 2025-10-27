import { proxyRequest } from '@/lib/proxy-helper';
import { NextRequest } from 'next/server';

export const dynamic = 'force-dynamic';

export async function POST(request: NextRequest, context: { params: Promise<{ action: string }> }) {
  const params = await context.params;
  const action = params.action;
  const body = await request.json();

  if (action !== 'start' && action !== 'stop') {
    return new Response(JSON.stringify({ error: 'Invalid action' }), { status: 400 });
  }

  return proxyRequest(`/admin/maintenance/${action}`,
    {
      method: 'POST',
      body: body,
    }
  );
}
