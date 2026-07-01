import ReactMarkdown from 'react-markdown'
import rehypeSanitize from 'rehype-sanitize'

interface MarkdownRendererProps {
  markdown: string
  className?: string
}

export default function MarkdownRenderer({ markdown, className }: MarkdownRendererProps) {
  return (
    <div className={className ?? 'prose prose-sm max-w-none text-brand-text'}>
      <ReactMarkdown rehypePlugins={[rehypeSanitize]}>
        {markdown}
      </ReactMarkdown>
    </div>
  )
}
