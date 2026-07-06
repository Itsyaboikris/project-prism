import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts"
import type { TrackedEvent } from "@/api/trackedEvents"
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import { occurrenceChartConfig } from "@/lib/chartConfig"

interface TrackedEventOccurrenceChartProps {
  trackedEvents: TrackedEvent[]
  onSelectEvent?: (key: string) => void
}

export function TrackedEventOccurrenceChart({
  trackedEvents,
  onSelectEvent,
}: TrackedEventOccurrenceChartProps) {
  const data = trackedEvents
    .filter((event) => event.occurrence_count > 0)
    .map((event) => ({
      event: event.name,
      key: event.key,
      count: event.occurrence_count,
    }))
    .sort((a, b) => b.count - a.count)

  if (data.length === 0) {
    return null
  }

  return (
    <ChartContainer config={occurrenceChartConfig} className="aspect-auto h-[220px] w-full">
      <BarChart
        data={data}
        layout="vertical"
        margin={{ top: 4, right: 8, left: 4, bottom: 4 }}
      >
        <CartesianGrid horizontal={false} strokeDasharray="3 3" />
        <XAxis type="number" tickLine={false} axisLine={false} allowDecimals={false} />
        <YAxis
          type="category"
          dataKey="event"
          tickLine={false}
          axisLine={false}
          width={120}
        />
        <ChartTooltip
          content={
            <ChartTooltipContent
              labelFormatter={(_, payload) => {
                const item = payload?.[0]?.payload as { event: string; key: string } | undefined
                if (!item) return ""
                return (
                  <span>
                    {item.event}{" "}
                    <span className="font-mono text-muted-foreground">({item.key})</span>
                  </span>
                )
              }}
            />
          }
        />
        <Bar
          dataKey="count"
          fill="var(--color-count)"
          radius={[0, 4, 4, 0]}
          cursor={onSelectEvent ? "pointer" : undefined}
          onClick={(barData) => {
            const key = (barData as { payload?: { key?: string } }).payload?.key
            if (key && onSelectEvent) onSelectEvent(key)
          }}
        />
      </BarChart>
    </ChartContainer>
  )
}
