import { NextRequest, NextResponse } from 'next/server';

export const dynamic = 'force-dynamic';

/**
 * [DEBUG] 特定の異常検知への回答を削除するAPI
 * 最小限のコードでルーティングが機能するかをテストします。
 */
export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id } = await params;
    console.log(`[DEBUG] DELETE request received for id: ${id}`);
    return NextResponse.json({ message: `[DEBUG] Received DELETE for id: ${id}` }, { status: 200 });
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    console.error(`[DEBUG] DELETE /anomaly-response/:id] Error:`, errorMessage);
    return NextResponse.json({ error: `An error occurred: ${errorMessage}` }, { status: 500 });
  }
}
