import { ExternalLink } from 'lucide-react';

export function Footer() {
  return (
    <section className="text-center py-6 border-t">
      <p className="text-sm text-gray-500 mb-2">
        HUNT - Highly Unified Needs Tracker
      </p>
      <div className="flex justify-center gap-4 text-xs text-gray-400">
        <span>v1.0.0</span>
        <span>•</span>
        <a
          href="https://github.com/KairiGit/hunt-chat-api"
          target="_blank"
          rel="noopener noreferrer"
          className="hover:text-blue-600 inline-flex items-center gap-1"
        >
          GitHub <ExternalLink className="h-3 w-3" />
        </a>
        <span>•</span>
        <span>MIT License</span>
      </div>
    </section>
  );
}
