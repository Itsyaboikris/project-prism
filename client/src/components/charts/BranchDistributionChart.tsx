import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts"
import type { ExperimentDashboardBranch } from "@/api/assignments"
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import { branchDistributionChartConfig } from "@/lib/chartConfig"
import { formatBranchWeightValue } from "@/lib/branchWeights"

interface BranchDistributionChartProps {
  branches: ExperimentDashboardBranch[]
}

function formatPercent(value: number) {
  return `${formatBranchWeightValue(value)}%`
}

export function BranchDistributionChart({ branches }: BranchDistributionChartProps) {
  if (branches.length === 0) {
    return (
      <p className="py-8 text-center text-sm text-muted-foreground">
        No branch data to chart yet.
      </p>
    )
  }

  const data = branches.map((branch) => ({
    branch: branch.branch_name,
    branchKey: branch.branch_key,
    configured: branch.configured_weight,
    actual: branch.assignment_share,
  }))

  return (
    <ChartContainer config={branchDistributionChartConfig} className="aspect-auto h-[280px] w-full">
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
                const item = payload?.[0]?.payload as { branch: string; branchKey: string } | undefined
                if (!item) return ""
                return (
                  <span>
                    {item.branch}{" "}
                    <span className="font-mono text-muted-foreground">({item.branchKey})</span>
                  </span>
                )
              }}
              formatter={(value) => formatPercent(Number(value))}
            />
          }
        />
        <ChartLegend content={<ChartLegendContent />} />
        <Bar dataKey="configured" fill="var(--color-configured)" radius={[4, 4, 0, 0]} />
        <Bar dataKey="actual" fill="var(--color-actual)" radius={[4, 4, 0, 0]} />
      </BarChart>
    </ChartContainer>
  )
}
