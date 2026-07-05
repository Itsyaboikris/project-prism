import { Link, useLocation } from "react-router-dom"
import { BarChart3, FlaskConical, ListTree, Zap } from "lucide-react"
import { cn } from "@/lib/utils"

const tabs = [
  { suffix: "", label: "Overview", icon: FlaskConical },
  { suffix: "/dashboard", label: "Dashboard", icon: BarChart3 },
  { suffix: "/assignments", label: "Assignments", icon: ListTree },
  { suffix: "/events", label: "Events", icon: Zap },
] as const

interface Props {
  appId: string
  experimentId: string
}

export function ExperimentNav({ appId, experimentId }: Props) {
  const { pathname } = useLocation()
  const base = `/applications/${appId}/experiments/${experimentId}`

  return (
    <nav className="flex gap-1 overflow-x-auto rounded-lg bg-muted/50 p-1">
      {tabs.map(({ suffix, label, icon: Icon }) => {
        const href = `${base}${suffix}`
        const isActive = suffix === "" ? pathname === base : pathname.startsWith(href)

        return (
          <Link
            key={suffix}
            to={href}
            className={cn(
              "inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium whitespace-nowrap transition-colors",
              isActive
                ? "bg-background text-foreground shadow-sm ring-1 ring-border"
                : "text-muted-foreground hover:text-foreground",
            )}
          >
            <Icon className="size-4 shrink-0 opacity-70" />
            {label}
          </Link>
        )
      })}
    </nav>
  )
}
