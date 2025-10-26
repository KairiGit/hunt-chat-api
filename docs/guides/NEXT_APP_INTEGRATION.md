# 外部Next.jsアプリケーションとの連携ガイド（ファイル流用編）

このドキュメントは、HUNT Chat-APIの既存フロントエンド資産を流用し、新しいNext.jsアプリケーションからGoバックエンドを安全に呼び出すための、より具体的な手順を解説します。

## 概要

基本的な考え方は、**既存のファイルをそのままコピー**し、必要な設定を行うことです。
特に、HUNT APIの認証情報を安全に管理するため、Next.jsのAPIルートをBFF（Backend for Frontend）として利用する構成をそのまま流用します。

## 前提条件

- Node.js v18以上
- 稼働中のHUNT Chat-API（Goバックエンド）インスタンス
- HUNT Chat-APIのAPIキー

## ステップ1: 新規Next.jsアプリケーションの作成

まず、連携させたいNext.jsアプリケーションを用意します。なければ新規に作成してください。

```bash
npx create-next-app@latest my-hunt-integration
cd my-hunt-integration
```

## ステップ2: APIルート（BFF）のコピー

Goバックエンドへのリクエストを中継し、APIキーを安全に管理するためのAPIルートをそのままコピーします。

1.  HUNTプロジェクトの `src/app/api/proxy` ディレクトリを、あなたのNext.jsアプリケーションの `src/app/api/` ディレクトリ配下にコピーします。
2.  `proxy` ディレクトリが依存しているヘルパー関数をコピーします。HUNTプロジェクトの `src/lib/proxy-helper.ts` を、あなたのプロジェクトの `src/lib/proxy-helper.ts` にコピーしてください。（`lib`ディレクトリがなければ作成してください）

**コピー後のディレクトリ構造（例）:**
```
my-hunt-integration/
└── src/
    ├── app/
    │   └── api/
    │       └── proxy/       # <- コピーしたディレクトリ
    │           ├── chat-input/
    │           │   └── route.ts
    │           └── ...
    └── lib/
        └── proxy-helper.ts  # <- コピーしたファイル
```

## ステップ3: 環境変数の設定

APIルートがGoバックエンドと通信するために必要な情報を環境変数として設定します。
Next.jsアプリケーションのルートに `.env.local` ファイルを作成してください。

```.env.local
# HUNT APIバックエンドのURL
HUNT_API_BASE_URL=http://localhost:8080

# HUNT APIの認証キー
HUNT_API_KEY=YOUR_SECRET_API_KEY
```

**【重要】**: `NEXT_PUBLIC_` プレフィックスは**絶対に付けないでください**。これらの変数はサーバーサイドでのみ使用され、クライアント（ブラウザ）に漏洩するのを防ぎます。

## ステップ4: フロントエンドのページとコンポーネントのコピー

利用したいページ（例: チャットページ）をそのままコピーします。

1.  HUNTプロジェクトの `src/app/chat/page.tsx` を、あなたのプロジェクトの `src/app/chat/page.tsx` にコピーします。
2.  `page.tsx` が依存しているコンポーネントやコンテキストも必要です。以下のディレクトリをHUNTプロジェクトから**そのまま上書きコピー**するのが最も簡単です。
    - `src/components/`
    - `src/contexts/`
    - `src/types/`

**コピー後のディレクトリ構造（例）:**
```
my-hunt-integration/
└── src/
    ├── app/
    │   └── chat/
    │       └── page.tsx       # <- コピーしたページ
    ├── components/            # <- コピーしたディレクトリ
    ├── contexts/              # <- コピーしたディレクトリ
    └── types/                 # <- コピーしたディレクトリ
```

## ステップ5: 依存ライブラリのインストール

コピーしたファイルが使用しているライブラリをインストールします。
HUNTプロジェクトの `package.json` を参考に、あなたのプロジェクトの `package.json` に以下の依存関係を追加し、インストールを実行してください。

**主な依存ライブラリ:**
- `lucide-react`
- `@radix-ui/...` （`@radix-ui/react-slot`など、`shadcn/ui`が使用するもの）
- `tailwindcss-animate`
- `zod`
- `react-hook-form`

```bash
npm install lucide-react @radix-ui/react-slot tailwindcss-animate zod react-hook-form
# その他、エラーに応じて適宜追加してください
```

また、`shadcn/ui` を利用しているため、`components.json` の設定や `globals.css` の内容もHUNTプロジェクトからコピーすることを推奨します。

## まとめ

以上の手順で、既存のHUNTフロントエンドの機能をあなたのNext.jsアプリケーションに移植できます。

- **API呼び出し**: フロントエンドコンポーネント → Next.jsのAPIルート(BFF) → Goバックエンド
- **認証**: BFF層で `X-API-KEY` ヘッダーを付与

この構成により、安全かつ効率的にHUNT APIを活用できます。