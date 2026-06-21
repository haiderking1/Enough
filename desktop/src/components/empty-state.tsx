interface EmptyStateProps {
  composer: React.ReactNode
}

export function EmptyState({ composer }: EmptyStateProps) {
  return (
    <div className="relative flex min-h-0 flex-1 flex-col">
      <div className="flex-1" />
      <div className="mx-auto w-full max-w-[720px] px-6 pb-4">{composer}</div>
    </div>
  )
}
