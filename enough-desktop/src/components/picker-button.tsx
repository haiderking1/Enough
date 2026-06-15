import { ChevronDown } from "lucide-react"
import { cn } from "../lib/utils"

interface PickerButtonProps {
  icon: React.ReactNode
  label: string
  open?: boolean
  disabled?: boolean
  onClick: () => void
  className?: string
}

export function PickerButton({ icon, label, open, disabled, onClick, className }: PickerButtonProps) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className={cn(
        "inline-flex max-w-[220px] items-center gap-1.5 rounded-full border border-border bg-surface/80 px-2.5 py-1 text-left transition-colors",
        "hover:border-border-strong hover:bg-surface disabled:cursor-not-allowed disabled:opacity-50",
        open && "border-border-strong bg-surface",
        className,
      )}
    >
      <span className="shrink-0 text-muted-foreground">{icon}</span>
      <span className="truncate text-[12px] font-medium text-foreground">{label}</span>
      <ChevronDown
        className={cn("h-3 w-3 shrink-0 text-muted-foreground transition-transform", open && "rotate-180")}
        strokeWidth={2}
      />
    </button>
  )
}
