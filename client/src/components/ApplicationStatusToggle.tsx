import type { ApplicationStatus } from "@/api/applications"

interface Props {
  status: ApplicationStatus
  disabled?: boolean
  onToggle: () => void
}

export function ApplicationStatusToggle({ status, disabled = false, onToggle }: Props) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={status === "active"}
      onClick={onToggle}
      className={`relative inline-flex h-7 w-12 shrink-0 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-slate-300 focus:ring-offset-2 ${status === "active" ? "bg-green-500" : "bg-slate-300"}`}
      disabled={disabled}
    >
      <span
        className={`inline-block size-5 rounded-full bg-white shadow-sm transition-transform ${status === "active" ? "translate-x-6" : "translate-x-1"}`}
      />
    </button>
  )
}
