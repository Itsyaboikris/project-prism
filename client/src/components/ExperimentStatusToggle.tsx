import type { ExperimentStatus } from "@/api/experiments"
import { cn } from "@/lib/utils"

interface Props {
  status: ExperimentStatus
  disabled?: boolean
  onToggle: () => void
}

export function ExperimentStatusToggle({ status, disabled = false, onToggle }: Props) {
  const isActive = status === "active"
  const isCompleted = status === "completed"

  return (
    <button
      type="button"
      role="switch"
      aria-checked={isActive}
      aria-label={
        isCompleted
          ? "Completed"
          : isActive
            ? "Turn experiment off"
            : "Turn experiment on"
      }
      onClick={onToggle}
      className={cn(
        "relative inline-flex h-7 w-12 shrink-0 items-center rounded-full transition-colors focus:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/30",
        isActive ? "bg-emerald-500" : "bg-muted-foreground/30",
      )}
      disabled={disabled || isCompleted}
      title={isCompleted ? "Completed experiments cannot be toggled." : undefined}
    >
      <span
        className={cn(
          "inline-block size-5 rounded-full bg-card shadow-sm transition-transform",
          isActive ? "translate-x-6" : "translate-x-1",
        )}
      />
    </button>
  )
}
