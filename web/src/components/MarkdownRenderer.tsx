import ReactMarkdown, { type Components } from 'react-markdown'
import rehypeSanitize from 'rehype-sanitize'

interface MarkdownRendererProps {
  markdown: string
  className?: string
}

// Explizites Element-Mapping statt @tailwindcss/typography (nicht installiert).
// Hält uns an brand-* Tokens und vermeidet eine zusätzliche Dependency.
const components: Components = {
  h1: ({ children }) => <h1 className="text-xl font-bold text-brand-text mt-4 mb-2 first:mt-0">{children}</h1>,
  h2: ({ children }) => <h2 className="text-lg font-semibold text-brand-text mt-4 mb-2 first:mt-0">{children}</h2>,
  h3: ({ children }) => <h3 className="text-base font-semibold text-brand-text mt-3 mb-1.5 first:mt-0">{children}</h3>,
  h4: ({ children }) => <h4 className="text-sm font-semibold text-brand-text mt-3 mb-1 first:mt-0">{children}</h4>,
  p: ({ children }) => <p className="text-sm text-brand-text my-2 leading-relaxed">{children}</p>,
  ul: ({ children }) => <ul className="list-disc pl-5 my-2 text-sm text-brand-text space-y-1">{children}</ul>,
  ol: ({ children }) => <ol className="list-decimal pl-5 my-2 text-sm text-brand-text space-y-1">{children}</ol>,
  li: ({ children }) => <li className="leading-relaxed">{children}</li>,
  a: ({ children, href }) => (
    <a href={href} className="text-brand-info underline underline-offset-2 hover:text-brand-text">
      {children}
    </a>
  ),
  strong: ({ children }) => <strong className="font-semibold text-brand-text">{children}</strong>,
  em: ({ children }) => <em className="italic">{children}</em>,
  code: ({ children }) => (
    <code className="font-mono text-xs bg-brand-border-subtle text-brand-text px-1 py-0.5 rounded">
      {children}
    </code>
  ),
  pre: ({ children }) => (
    <pre className="font-mono text-xs bg-brand-border-subtle text-brand-text p-3 rounded overflow-x-auto my-2">
      {children}
    </pre>
  ),
  blockquote: ({ children }) => (
    <blockquote className="border-l-4 border-brand-yellow pl-3 my-2 text-brand-text-muted italic">
      {children}
    </blockquote>
  ),
  hr: () => <hr className="my-4 border-brand-border-subtle" />,
  img: ({ src, alt }) => (
    <img src={typeof src === 'string' ? src : undefined} alt={alt ?? ''} className="max-w-full h-auto rounded my-2" />
  ),
}

export default function MarkdownRenderer({ markdown, className }: MarkdownRendererProps) {
  return (
    <div className={className ?? 'text-brand-text'}>
      <ReactMarkdown rehypePlugins={[rehypeSanitize]} components={components}>
        {markdown}
      </ReactMarkdown>
    </div>
  )
}
