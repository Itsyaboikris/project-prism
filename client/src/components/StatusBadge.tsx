import type { ExperimentStatus } from "@/api/experiments"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

const styles: Record<ExperimentStatus, string> = {
  draft: "border-border bg-muted/50 text-muted-foreground",
  active: "border-emerald-500/20 bg-emerald-500/10 text-emerald-400",
  paused: "border-amber-500/20 bg-amber-500/10 text-amber-400",
  completed: "border-sky-500/20 bg-sky-500/10 text-sky-400",
}

interface Props {
  status: ExperimentStatus
}

export function StatusBadge({ status }: Props) {
  return (
    <Badge variant="outline" className={cn("capitalize", styles[status])}>
      {status}
    </Badge>
  )
}
