import { marked } from "marked"
import { memo, useMemo } from "react"
import ReactMarkdown from "react-markdown"
import remarkGfm from "remark-gfm"
import remarkBreaks from "remark-breaks"
import { cn } from "../lib/utils"

interface MarkdownContentProps {
  /** Stable per message/block — required for block-level memoization. */
  id: string
  text: string
  className?: string
  streaming?: boolean
}

const remarkPlugins = [remarkGfm, remarkBreaks]

const markdownComponents = {
  p: ({ children }: { children?: React.ReactNode }) => <p className="mb-3 last:mb-0">{children}</p>,
  strong: ({ children }: { children?: React.ReactNode }) => (
    <strong className="font-semibold text-foreground">{children}</strong>
  ),
  em: ({ children }: { children?: React.ReactNode }) => (
    <em className="italic text-foreground/80">{children}</em>
  ),
  a: ({ href, children }: { href?: string; children?: React.ReactNode }) => (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="text-accent underline decoration-accent/40 underline-offset-2 hover:decoration-accent"
    >
      {children}
    </a>
  ),
  ul: ({ children }: { children?: React.ReactNode }) => (
    <ul className="mb-3 list-disc space-y-1 pl-5 last:mb-0">{children}</ul>
  ),
  ol: ({ children }: { children?: React.ReactNode }) => (
    <ol className="mb-3 list-decimal space-y-1 pl-5 last:mb-0">{children}</ol>
  ),
  li: ({ children }: { children?: React.ReactNode }) => <li className="pl-0.5">{children}</li>,
  blockquote: ({ children }: { children?: React.ReactNode }) => (
    <blockquote className="mb-3 border-l-2 border-border-strong pl-3 text-muted-foreground last:mb-0">
      {children}
    </blockquote>
  ),
  hr: () => <hr className="my-4 border-border" />,
  h1: ({ children }: { children?: React.ReactNode }) => (
    <h1 className="mb-2 mt-4 text-xl font-semibold text-foreground first:mt-0">{children}</h1>
  ),
  h2: ({ children }: { children?: React.ReactNode }) => (
    <h2 className="mb-2 mt-4 text-lg font-semibold text-foreground first:mt-0">{children}</h2>
  ),
  h3: ({ children }: { children?: React.ReactNode }) => (
    <h3 className="mb-2 mt-3 text-base font-semibold text-foreground first:mt-0">{children}</h3>
  ),
  h4: ({ children }: { children?: React.ReactNode }) => (
    <h4 className="mb-2 mt-3 text-sm font-semibold text-foreground first:mt-0">{children}</h4>
  ),
  pre: ({ children }: { children?: React.ReactNode }) => (
    <pre className="mb-3 overflow-x-auto rounded-lg border border-border bg-surface px-3 py-2.5 last:mb-0">
      {children}
    </pre>
  ),
  code: ({ className: codeClass, children }: { className?: string; children?: React.ReactNode }) => {
    const isBlock = Boolean(codeClass?.includes("language-"))
    if (isBlock) {
      return (
        <code className={cn("block font-mono text-[13px] leading-relaxed text-foreground/90", codeClass)}>
          {children}
        </code>
      )
    }
    return (
      <code className="rounded bg-surface px-1.5 py-0.5 font-mono text-[13px] text-foreground/90">
        {children}
      </code>
    )
  },
  table: ({ children }: { children?: React.ReactNode }) => (
    <div className="mb-3 overflow-x-auto last:mb-0">
      <table className="w-full border-collapse text-[13px]">{children}</table>
    </div>
  ),
  thead: ({ children }: { children?: React.ReactNode }) => (
    <thead className="border-b border-border-strong">{children}</thead>
  ),
  th: ({ children }: { children?: React.ReactNode }) => (
    <th className="px-3 py-2 text-left font-semibold text-foreground">{children}</th>
  ),
  td: ({ children }: { children?: React.ReactNode }) => (
    <td className="border-t border-border px-3 py-2 text-foreground/85">{children}</td>
  ),
}

function parseMarkdownIntoBlocks(markdown: string): string[] {
  if (!markdown) return []
  const tokens = marked.lexer(markdown, { gfm: true })
  return tokens.map((token) => token.raw)
}

const MarkdownBlock = memo(
  function MarkdownBlock({ content }: { content: string }) {
    return (
      <ReactMarkdown remarkPlugins={remarkPlugins} components={markdownComponents}>
        {content}
      </ReactMarkdown>
    )
  },
  (prev, next) => prev.content === next.content,
)

export const MarkdownContent = memo(function MarkdownContent({
  id,
  text,
  className,
  streaming,
}: MarkdownContentProps) {
  const blocks = useMemo(() => parseMarkdownIntoBlocks(text), [text])

  return (
    <div className={cn("markdown-content text-[14px] leading-relaxed text-foreground/90", className)}>
      {blocks.map((block, index) => (
        <MarkdownBlock key={`${id}-block_${index}`} content={block} />
      ))}
      {streaming && (
        <span className="ml-0.5 inline-block h-[1.05em] w-[2px] translate-y-[2px] bg-accent align-text-bottom animate-caret" />
      )}
    </div>
  )
})
