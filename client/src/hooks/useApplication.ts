import { useEffect, useState } from "react"
import { applicationsApi, type Application } from "@/api/applications"
import { ApiError } from "@/api/client"

export function useApplication(appId: string | undefined) {
  const [app, setApp] = useState<Application | null>(null)
  const [loading, setLoading] = useState(Boolean(appId))
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!appId) {
      setApp(null)
      setLoading(false)
      return
    }

    setLoading(true)
    setError(null)

    applicationsApi
      .get(appId)
      .then(setApp)
      .catch((err) =>
        setError(err instanceof ApiError ? err.message : "Failed to load application"),
      )
      .finally(() => setLoading(false))
  }, [appId])

  return { app, loading, error }
}
