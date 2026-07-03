import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"
import ApplicationsPage from "@/pages/ApplicationsPage"
import ApplicationDetailPage from "@/pages/ApplicationDetailPage"

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Navigate to="/applications" replace />} />
        <Route path="/applications" element={<ApplicationsPage />} />
        <Route path="/applications/:id" element={<ApplicationDetailPage />} />
      </Routes>
    </BrowserRouter>
  )
}
