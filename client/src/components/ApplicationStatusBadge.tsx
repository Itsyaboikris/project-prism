import type { ApplicationStatus } from "@/api/applications"

const styles: Record<ApplicationStatus, string> = {
  active: "bg-green-100 text-green-700",
  inactive: "bg-slate-200 text-slate-700",
}

interface Props {
  status: ApplicationStatus
}

export function ApplicationStatusBadge({ status }: Props) {
  return (
    <span
      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium capitalize ${styles[status]}`}
    >
      {status}
    </span>
  )
}
