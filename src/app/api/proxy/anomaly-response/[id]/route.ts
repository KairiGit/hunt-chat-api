import { proxyRequest } from '@/lib/proxy-helper';
import { NextRequest, NextResponse } from 'next/server';

export const dynamic = 'force-dynamic';

/**
 * 特定の異常検知への回答を削除するプロキシAPI
 * @param request NextRequest
 * @param params URLから取得したパラメータ（idを含む）
 */
export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    // Note: このプロジェクトのNext.jsのバージョンでは、App Routerの`params`がPromiseとして渡される
    // 型定義に従い、awaitで解決する必要がある
    const { id } = await params;
    
    if (!id) {
      return NextResponse.json(
        { success: false, error: 'IDが指定されていません' },
        { status: 400 }
      );
    }

    const endpoint = `/api/v1/ai/anomaly-response/${id}`;

    console.log(`[Proxy DELETE] Forwarding to: ${endpoint}`);

    // proxyRequestヘルパーを使ってリクエストを転送
    return await proxyRequest(endpoint, {
      method: 'DELETE',
    });

  } catch (error) {
    // `await params` や予期せぬエラーをキャッチする
    console.error(`[Proxy DELETE /anomaly-response/:id] Top-level error:`, error);
    
    const errorMessage = error instanceof Error ? error.message : '不明なエラーです';

    return NextResponse.json(
      { success: false, error: `プロキシ処理中に予期せぬエラーが発生しました: ${errorMessage}` },
      { status: 500 }
    );
  }
}