import type { ExperimentStatus } from "@/api/experiments"

interface Props {
  status: ExperimentStatus
  disabled?: boolean
  onToggle: () => void
}

export function ExperimentStatusToggle({ status, disabled = false, onToggle }: Props) {
  const isActive = status === "active"
  const isCompleted = status === "completed"

  return (
    <div className="flex items-center gap-2">
      <span className="text-xs font-medium uppercase tracking-wide text-slate-500">
        {isCompleted ? "Completed" : isActive ? "Active" : "Off"}
      </span>
      <button
        type="button"
        role="switch"
        aria-checked={isActive}
        aria-label={isActive ? "Turn experiment off" : "Turn experiment on"}
        onClick={onToggle}
        className={`relative inline-flex h-7 w-12 shrink-0 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-slate-300 focus:ring-offset-2 ${
          isActive ? "bg-green-500" : "bg-slate-300"
        }`}
        disabled={disabled || isCompleted}
        title={isCompleted ? "Completed experiments cannot be toggled." : undefined}
      >
        <span
          className={`inline-block size-5 rounded-full bg-white shadow-sm transition-transform ${
            isActive ? "translate-x-6" : "translate-x-1"
          }`}
        />
      </button>
    </div>
  )
}
