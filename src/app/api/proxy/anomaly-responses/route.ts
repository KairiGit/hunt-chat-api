import { proxyRequest } from '@/lib/proxy-helper';

export const dynamic = 'force-dynamic';

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  
  return proxyRequest('/api/v1/ai/anomaly-responses', {
    method: 'GET',
    searchParams,
  });
}

/**
 * [WORKAROUND] Deletes anomaly responses.
 * - If an 'id' query param is present, deletes a single response.
 * - Otherwise, deletes all responses.
 * This avoids using a dynamic route ([id]) which causes issues on Vercel.
 */
export async function DELETE(request: Request) {
  const { searchParams } = new URL(request.url);
  const id = searchParams.get('id');

  if (id) {
    // Individual deletion
    console.log(`[Proxy DELETE] Forwarding individual delete for id: ${id}`);
    const endpoint = `/api/v1/ai/anomaly-response/${id}`;
    return proxyRequest(endpoint, {
      method: 'DELETE',
    });
  } else {
    // Delete all
    console.log(`[Proxy DELETE] Forwarding delete all request.`);
    const endpoint = '/api/v1/ai/anomaly-responses';
    return proxyRequest(endpoint, {
      method: 'DELETE',
    });
  }
}