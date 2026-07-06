import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts"
import type { ExperimentDashboardBranch } from "@/api/assignments"
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import { conversionChartConfig } from "@/lib/chartConfig"
import { formatBranchWeightValue } from "@/lib/branchWeights"

interface ConversionByBranchChartProps {
  branches: ExperimentDashboardBranch[]
  eventName: string
}

function formatPercent(value: number) {
  return `${formatBranchWeightValue(value)}%`
}

export function ConversionByBranchChart({ branches, eventName }: ConversionByBranchChartProps) {
  const data = branches.map((branch) => ({
    branch: branch.branch_name,
    branchKey: branch.branch_key,
    conversion: branch.conversion_rate ?? 0,
    uniqueUsers: branch.unique_event_users ?? 0,
    eventCount: branch.event_count ?? 0,
  }))

  const hasConversions = data.some((item) => item.conversion > 0)

  if (!hasConversions) {
    return (
      <p className="py-8 text-center text-sm text-muted-foreground">
        No {eventName} conversions recorded yet.
      </p>
    )
  }

  return (
    <ChartContainer config={conversionChartConfig} className="aspect-auto h-[240px] w-full">
      <BarChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
        <CartesianGrid vertical={false} strokeDasharray="3 3" />
        <XAxis
          dataKey="branch"
          tickLine={false}
          axisLine={false}
          tickMargin={8}
        />
        <YAxis
          tickLine={false}
          axisLine={false}
          tickMargin={8}
          tickFormatter={(value) => `${value}%`}
          domain={[0, 100]}
        />
        <ChartTooltip
          content={
            <ChartTooltipContent
              labelFormatter={(_, payload) => {
                const item = payload?.[0]?.payload as
                  | { branch: string; branchKey: string; uniqueUsers: number; eventCount: number }
                  | undefined
                if (!item) return ""
                return (
                  <div className="space-y-1">
                    <div>
                      {item.branch}{" "}
                      <span className="font-mono text-muted-foreground">({item.branchKey})</span>
                    </div>
                    <div className="text-muted-foreground">
                      {item.uniqueUsers} unique users · {item.eventCount} events
                    </div>
                  </div>
                )
              }}
              formatter={(value) => formatPercent(Number(value))}
            />
          }
        />
        <Bar dataKey="conversion" fill="var(--color-conversion)" radius={[4, 4, 0, 0]} />
      </BarChart>
    </ChartContainer>
  )
}
