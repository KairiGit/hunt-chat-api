import React, { useEffect } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import { cn } from '@/lib/utils';
import mermaid from 'mermaid';

// Mermaidの初期設定
mermaid.initialize({
  startOnLoad: false, // 自動実行はしない
  theme: 'neutral', // 'dark', 'forest' なども選択可能
  securityLevel: 'loose',
});

interface MarkdownProps {
  children: string;
  className?: string;
}

export function Markdown({ children, className }: MarkdownProps) {
  // childrenが変更されたらMermaidを実行
  useEffect(() => {
    try {
      mermaid.run({
        nodes: document.querySelectorAll('.language-mermaid'),
      });
    } catch (e) {
      console.error('Mermaid rendering error:', e);
    }
  }, [children]);

  return (
    <div className={cn('prose prose-sm max-w-none', className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeRaw]}
        components={{
        // 見出し
        h1: ({ children }) => (
          <h1 className="text-2xl font-bold mt-6 mb-4 text-purple-900 dark:text-purple-100 border-b-2 border-purple-200 dark:border-purple-800 pb-2">
            {children}
          </h1>
        ),
        h2: ({ children }) => (
          <h2 className="text-xl font-bold mt-5 mb-3 text-purple-800 dark:text-purple-200 border-b border-purple-200 dark:border-purple-800 pb-1">
            {children}
          </h2>
        ),
        h3: ({ children }) => (
          <h3 className="text-lg font-semibold mt-4 mb-2 text-purple-700 dark:text-purple-300">
            {children}
          </h3>
        ),
        h4: ({ children }) => (
          <h4 className="text-base font-semibold mt-3 mb-2 text-purple-600 dark:text-purple-400">
            {children}
          </h4>
        ),
        
        // 段落
        p: ({ children }) => (
          <p className="mb-3 leading-relaxed text-purple-900 dark:text-purple-100">
            {children}
          </p>
        ),
        
        // リスト
        ul: ({ children }) => (
          <ul className="list-disc list-inside mb-3 space-y-1 ml-2">
            {children}
          </ul>
        ),
        ol: ({ children }) => (
          <ol className="list-decimal list-inside mb-3 space-y-1 ml-2">
            {children}
          </ol>
        ),
        li: ({ children }) => (
          <li className="text-purple-900 dark:text-purple-100 ml-2">
            {children}
          </li>
        ),
        
        // コードブロック
        code: ({ inline, className, children, ...props }: {
          inline?: boolean;
          className?: string;
          children?: React.ReactNode;
        }) => {
          const match = /language-(\w+)/.exec(className || '');
          const lang = match && match[1];

          if (lang === 'mermaid') {
            return (
              <pre className="language-mermaid" {...props}>
                {String(children)}
              </pre>
            );
          }

          return inline ? (
            <code
              className="px-1.5 py-0.5 rounded bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-200 font-mono text-sm"
              {...props}
            >
              {children}
            </code>
          ) : (
            <pre className="bg-purple-50 dark:bg-purple-950/50 rounded-lg p-4 overflow-x-auto mb-3 border border-purple-200 dark:border-purple-800">
              <code
                className={cn('font-mono text-sm text-purple-900 dark:text-purple-100', className)}
                {...props}
              >
                {children}
              </code>
            </pre>
          );
        },
        
        // テーブル
        table: ({ children }) => (
          <div className="overflow-x-auto mb-3">
            <table className="min-w-full divide-y divide-purple-200 dark:divide-purple-800 border border-purple-200 dark:border-purple-800 rounded-lg">
              {children}
            </table>
          </div>
        ),
        thead: ({ children }) => (
          <thead className="bg-purple-100 dark:bg-purple-900/30">
            {children}
          </thead>
        ),
        tbody: ({ children }) => (
          <tbody className="divide-y divide-purple-200 dark:divide-purple-800 bg-white dark:bg-transparent">
            {children}
          </tbody>
        ),
        tr: ({ children }) => (
          <tr className="hover:bg-purple-50 dark:hover:bg-purple-900/20 transition-colors">
            {children}
          </tr>
        ),
        th: ({ children }) => (
          <th className="px-4 py-2 text-left text-sm font-semibold text-purple-900 dark:text-purple-100">
            {children}
          </th>
        ),
        td: ({ children }) => (
          <td className="px-4 py-2 text-sm text-purple-900 dark:text-purple-100">
            {children}
          </td>
        ),
        
        // 引用
        blockquote: ({ children }) => (
          <blockquote className="border-l-4 border-purple-400 dark:border-purple-600 pl-4 py-2 mb-3 italic text-purple-700 dark:text-purple-300 bg-purple-50 dark:bg-purple-950/30 rounded-r">
            {children}
          </blockquote>
        ),
        
        // 水平線
        hr: () => (
          <hr className="my-4 border-purple-200 dark:border-purple-800" />
        ),
        
        // リンク
        a: ({ children, href }) => (
          <a
            href={href}
            target="_blank"
            rel="noopener noreferrer"
            className="text-purple-600 dark:text-purple-400 hover:text-purple-800 dark:hover:text-purple-200 underline font-medium"
          >
            {children}
          </a>
        ),
        
        // 強調
        strong: ({ children }) => (
          <strong className="font-bold text-purple-900 dark:text-purple-100">
            {children}
          </strong>
        ),
        em: ({ children }) => (
          <em className="italic text-purple-800 dark:text-purple-200">
            {children}
          </em>
        ),
      }}
    >
      {children}
    </ReactMarkdown>
    </div>
  );
}
