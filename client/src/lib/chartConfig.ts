import type { ChartConfig } from "@/components/ui/chart"

export const branchDistributionChartConfig = {
  configured: {
    label: "Configured weight",
    color: "var(--chart-1)",
  },
  actual: {
    label: "Actual share",
    color: "var(--chart-2)",
  },
} satisfies ChartConfig

export const conversionChartConfig = {
  conversion: {
    label: "Conversion rate",
    color: "var(--chart-3)",
  },
} satisfies ChartConfig

export const occurrenceChartConfig = {
  count: {
    label: "Occurrences",
    color: "var(--chart-1)",
  },
} satisfies ChartConfig

export function chartColor(index: number) {
  return `var(--chart-${(index % 5) + 1})`
}
