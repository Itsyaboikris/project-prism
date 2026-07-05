import { Skeleton } from "@/components/ui/skeleton"

interface PageLoadingProps {
  rows?: number
}

export function PageLoading({ rows = 3 }: PageLoadingProps) {
  return (
    <div className="space-y-4">
      <Skeleton className="h-8 w-64" />
      <Skeleton className="h-10 w-full" />
      {Array.from({ length: rows }).map((_, i) => (
        <Skeleton key={i} className="h-16 w-full" />
      ))}
    </div>
  )
}

export function TableLoading({ rows = 5 }: PageLoadingProps) {
  return (
    <div className="space-y-3 p-6">
      <Skeleton className="h-8 w-full" />
      {Array.from({ length: rows }).map((_, i) => (
        <Skeleton key={i} className="h-10 w-full" />
      ))}
    </div>
  )
}
