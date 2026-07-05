import { api } from "./client"
import type { ExperimentStatus } from "./experiments"

export interface ExperimentAssignmentListItem {
  id: string
  application_id: string
  experiment_id: string
  branch_id: string
  user_id: string
  assigned_at: string
  context_json: unknown | null
  created_at: string
  updated_at: string
  branch_key: string
  branch_name: string
  branch_weight: number
}

export interface ExperimentAssignmentsView {
  experiment_id: string
  experiment_key: string
  experiment_name: string
  experiment_status: ExperimentStatus
  assignments: ExperimentAssignmentListItem[]
}

export interface ExperimentDashboardBranch {
  branch_id: string
  branch_key: string
  branch_name: string
  configured_weight: number
  assignment_count: number
  assignment_share: number
  event_count?: number
  unique_event_users?: number
  conversion_rate?: number
}

export interface ExperimentDashboard {
  experiment_id: string
  experiment_key: string
  experiment_name: string
  experiment_status: ExperimentStatus
  event_name?: string
  total_assignments: number
  branch_count: number
  branches: ExperimentDashboardBranch[]
}

export const assignmentsApi = {
  listByExperiment: (appId: string, experimentId: string) =>
    api.get<ExperimentAssignmentsView>(
      `/api/v1/applications/${appId}/experiments/${experimentId}/assignments`,
    ),

  getExperimentDashboard: (appId: string, experimentId: string, eventName?: string) => {
    const query = eventName ? `?event_name=${encodeURIComponent(eventName)}` : ""
    return api.get<ExperimentDashboard>(
      `/api/v1/applications/${appId}/experiments/${experimentId}/dashboard${query}`,
    )
  },
}
