import type { ApplicationStatus } from "@/api/applications"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

const styles: Record<ApplicationStatus, string> = {
  active: "border-emerald-500/20 bg-emerald-500/10 text-emerald-400",
  inactive: "border-border bg-muted/50 text-muted-foreground",
}

interface Props {
  status: ApplicationStatus
}

export function ApplicationStatusBadge({ status }: Props) {
  return (
    <Badge variant="outline" className={cn("capitalize", styles[status])}>
      {status}
    </Badge>
  )
}
