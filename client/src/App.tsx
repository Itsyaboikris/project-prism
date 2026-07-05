import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"
import RequireAdmin from "@/auth/RequireAdmin"
import AdminShell from "@/components/AdminShell"
import ApplicationsPage from "@/pages/ApplicationsPage"
import ApplicationDetailPage from "@/pages/ApplicationDetailPage"
import AdminUsersPage from "@/pages/AdminUsersPage"
import ActivateInvitePage from "@/pages/ActivateInvitePage"
import CreateExperimentPage from "@/pages/CreateExperimentPage"
import ExperimentsPage from "@/pages/ExperimentsPage"
import ExperimentDetailPage from "@/pages/ExperimentDetailPage"
import ExperimentAssignmentsPage from "@/pages/ExperimentAssignmentsPage"
import ExperimentDashboardPage from "@/pages/ExperimentDashboardPage"
import ExperimentEventsPage from "@/pages/ExperimentEventsPage"
import LoginPage from "@/pages/LoginPage"

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/activate" element={<ActivateInvitePage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route element={<RequireAdmin />}>
          <Route element={<AdminShell />}>
            <Route path="/" element={<Navigate to="/applications" replace />} />
            <Route path="/applications" element={<ApplicationsPage />} />
            <Route path="/applications/:id" element={<ApplicationDetailPage />} />
            <Route path="/applications/:appId/experiments" element={<ExperimentsPage />} />
            <Route path="/applications/:appId/experiments/new" element={<CreateExperimentPage />} />
            <Route path="/applications/:appId/experiments/:id" element={<ExperimentDetailPage />} />
            <Route
              path="/applications/:appId/experiments/:id/assignments"
              element={<ExperimentAssignmentsPage />}
            />
            <Route
              path="/applications/:appId/experiments/:id/dashboard"
              element={<ExperimentDashboardPage />}
            />
            <Route
              path="/applications/:appId/experiments/:id/events"
              element={<ExperimentEventsPage />}
            />
            <Route path="/admin/users" element={<AdminUsersPage />} />
          </Route>
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
