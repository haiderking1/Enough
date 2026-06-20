interface EmptyStateProps {
  cwd: string
  composer: React.ReactNode
}

export function EmptyState({ cwd, composer }: EmptyStateProps) {
  return (
    <div className="relative flex min-h-0 flex-1 flex-col">
      <div className="flex-1" />

      <div className="w-full px-6 pb-4">
        <div className="mb-3 flex flex-wrap items-center gap-2">
          <Chip icon={<LaptopIcon />} label="Local" />
          <Chip icon={<FolderIcon />} label={cwd} />
        </div>
        {composer}
      </div>
    </div>
  )
}

function Chip({ icon, label }: { icon: React.ReactNode; label: string }) {
  return (
    <div className="flex items-center gap-2 rounded-full border border-border-strong bg-surface px-3 py-1.5 text-[13px] text-foreground/90">
      <span className="text-muted-foreground">{icon}</span>
      <span className="max-w-[240px] truncate">{label}</span>
    </div>
  )
}

function LaptopIcon() {
  return (
    <svg className="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">
      <rect x="3" y="5" width="18" height="12" rx="1.5" />
      <path d="M2 19h20" />
    </svg>
  )
}

function FolderIcon() {
  return (
    <svg className="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">
      <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7z" />
    </svg>
  )
}
