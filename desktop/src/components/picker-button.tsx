import { ChevronDown } from "lucide-react"
import { cn } from "../lib/utils"

interface PickerButtonProps {
  icon?: React.ReactNode
  label: string
  open?: boolean
  disabled?: boolean
  onClick: () => void
  className?: string
}

// Cursor-style inline trigger: plain muted text + chevron, no pill. Sits inside
// the composer's input row. The whole control lightens on hover.
export function PickerButton({ icon, label, open, disabled, onClick, className }: PickerButtonProps) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className={cn(
        "inline-flex max-w-[220px] shrink-0 items-center gap-1.5 rounded-md px-1.5 py-1 text-left text-muted-foreground transition-colors hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50",
        className,
      )}
    >
      {icon && <span className="shrink-0">{icon}</span>}
      <span className="truncate text-xs font-medium">{label}</span>
      <ChevronDown
        className={cn("h-3 w-3 shrink-0 transition-transform", open && "rotate-180")}
        strokeWidth={2}
      />
    </button>
  )
}