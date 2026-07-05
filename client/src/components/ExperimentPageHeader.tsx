import { PageHeader, type BreadcrumbItem } from "@/components/PageHeader"
import { ExperimentNav } from "@/components/ExperimentNav"

interface Props {
  appId: string
  experimentId: string
  title: string
  description?: string
  appName?: string
  actions?: React.ReactNode
}

export function ExperimentPageHeader({
  appId,
  experimentId,
  title,
  description,
  appName = "Application",
  actions,
}: Props) {
  const breadcrumbs: BreadcrumbItem[] = [
    { label: "Applications", href: "/applications" },
    { label: appName, href: `/applications/${appId}` },
    { label: "Experiments", href: `/applications/${appId}/experiments` },
    { label: title },
  ]

  return (
    <PageHeader
      title={title}
      description={description}
      breadcrumbs={breadcrumbs}
      actions={actions}
    >
      <ExperimentNav appId={appId} experimentId={experimentId} />
    </PageHeader>
  )
}
