import { NextRequest, NextResponse } from 'next/server';

const GO_API_BASE_URL = 'http://localhost:8080/api/v1';

export async function POST(request: NextRequest) {
  try {
    const formData = await request.formData();

    const response = await fetch(`${GO_API_BASE_URL}/ai/analyze-file`, {
      method: 'POST',
      body: formData,
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Failed to parse error response' }));
      return NextResponse.json({ error: 'Failed to forward request to Go API', details: errorData }, { status: response.status });
    }

    const data = await response.json();
    return NextResponse.json(data);

  } catch (error) {
    console.error('Internal Server Error:', error);
    return NextResponse.json({ error: 'Internal Server Error' }, { status: 500 });
  }
}
