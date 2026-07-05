import { NavLink, Outlet } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { useAuth } from "@/auth/AuthContext"

function linkClassName(isActive: boolean) {
  return [
    "rounded-lg border px-3 py-1.5 text-sm shadow-sm transition-colors",
    isActive
      ? "border-slate-900 bg-slate-900 text-white"
      : "border-slate-200 bg-white text-slate-700 hover:border-slate-300 hover:text-slate-900",
  ].join(" ")
}

export default function AdminShell() {
  const { user, logout } = useAuth()

  return (
    <>
      <div className="fixed top-4 right-4 z-20 flex flex-wrap items-center justify-end gap-2">
        <span className="hidden rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm text-slate-500 shadow-sm sm:inline-flex">
          {user?.email}
        </span>
        <NavLink to="/applications" className={({ isActive }) => linkClassName(isActive)}>
          Applications
        </NavLink>
        <NavLink to="/admin/users" className={({ isActive }) => linkClassName(isActive)}>
          Users
        </NavLink>
        <Button type="button" variant="outline" onClick={() => void logout()}>
          Logout
        </Button>
      </div>
      <Outlet />
    </>
  )
}
