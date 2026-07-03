import type { ExperimentStatus } from "@/api/experiments"

const styles: Record<ExperimentStatus, string> = {
  draft: "bg-slate-100 text-slate-600",
  active: "bg-green-100 text-green-700",
  paused: "bg-yellow-100 text-yellow-700",
  completed: "bg-blue-100 text-blue-700",
}

interface Props {
  status: ExperimentStatus
}

export function StatusBadge({ status }: Props) {
  return (
    <span
      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium capitalize ${styles[status]}`}
    >
      {status}
    </span>
  )
}
