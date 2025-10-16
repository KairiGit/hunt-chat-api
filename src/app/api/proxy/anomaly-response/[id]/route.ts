import { proxyRequest } from '@/lib/proxy-helper';
import { NextRequest, NextResponse } from 'next/server';

export const dynamic = 'force-dynamic';

/**
 * [WORKAROUND] 特定の異常検知への回答を削除するプロキシAPI
 * Vercel/Next.jsの動的ルートでのDELETEメソッドの問題を回避するため、POSTメソッドで削除を処理します。
 */
export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id } = await params;
    
    if (!id) {
      return NextResponse.json(
        { success: false, error: 'IDが指定されていません' },
        { status: 400 }
      );
    }

    const endpoint = `/api/v1/ai/anomaly-response/${id}`;

    console.log(`[Proxy POST-as-DELETE] Forwarding to: DELETE ${endpoint}`);

    // proxyRequestヘルパーを使って、バックエンドにはDELETEとしてリクエストを転送
    return await proxyRequest(endpoint, {
      method: 'DELETE',
    });

  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : '不明なエラーです';
    console.error(`[Proxy POST-as-DELETE /anomaly-response/:id] Top-level error:`, errorMessage);

    return NextResponse.json(
      { success: false, error: `プロキシ処理中に予期せぬエラーが発生しました: ${errorMessage}` },
      { status: 500 }
    );
  }
}