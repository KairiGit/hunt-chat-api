import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async rewrites() {
    // ローカル開発環境のみ localhost:8080 にリライト
    if (process.env.NODE_ENV === 'development') {
      return [
        {
          source: '/api/proxy/:path*',
          destination: 'http://localhost:8080/api/v1/:path*',
        },
        {
          source: '/api/v1/:path*',
          destination: 'http://localhost:8080/api/v1/:path*',
        },
      ];
    }
    // 本番環境では Next.js のリライトを使用せず、vercel.json のルーティングに任せる
    return [];
  },
};

export default nextConfig;
