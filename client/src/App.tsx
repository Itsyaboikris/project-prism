import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"
import ApplicationsPage from "@/pages/ApplicationsPage"
import ApplicationDetailPage from "@/pages/ApplicationDetailPage"
import ExperimentsPage from "@/pages/ExperimentsPage"
import ExperimentDetailPage from "@/pages/ExperimentDetailPage"
import ExperimentAssignmentsPage from "@/pages/ExperimentAssignmentsPage"
import ExperimentDashboardPage from "@/pages/ExperimentDashboardPage"

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Navigate to="/applications" replace />} />
        <Route path="/applications" element={<ApplicationsPage />} />
        <Route path="/applications/:id" element={<ApplicationDetailPage />} />
        <Route path="/applications/:appId/experiments" element={<ExperimentsPage />} />
        <Route path="/applications/:appId/experiments/:id" element={<ExperimentDetailPage />} />
        <Route
          path="/applications/:appId/experiments/:id/assignments"
          element={<ExperimentAssignmentsPage />}
        />
        <Route
          path="/applications/:appId/experiments/:id/dashboard"
          element={<ExperimentDashboardPage />}
        />
      </Routes>
    </BrowserRouter>
  )
}
