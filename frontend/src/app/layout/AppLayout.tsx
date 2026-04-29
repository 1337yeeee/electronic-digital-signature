import { NavLink, Outlet } from "react-router-dom";
import { useAuth } from "../../auth/AuthContext";

export function AppLayout() {
  const { currentUser, logout, authNotice, clearAuthNotice } = useAuth();

  return (
    <main className="app-shell">
      <aside className="app-sidebar">
        <p className="eyebrow">EDS Lab</p>
        <h1>Web console</h1>
        <p className="sidebar-copy">
          Authenticated workspace for signatures, users, and document flows.
        </p>

        <nav className="app-nav">
          <NavLink to="/app" end>
            Overview
          </NavLink>
          <NavLink to="/app/profile">
            Profile
          </NavLink>
          <NavLink to="/app/documents" end>
            My Documents
          </NavLink>
          <NavLink to="/app/documents/flow">
            Document Flow
          </NavLink>
          <NavLink to="/app/server-signed-message">
            Server Signed
          </NavLink>
          <NavLink to="/app/signatures/verify">
            Verify Signature
          </NavLink>
        </nav>

        <div className="user-chip">
          <span className="meta-label">Signed in as</span>
          <strong>{currentUser?.name ?? currentUser?.email ?? "Unknown user"}</strong>
          <small>{currentUser?.email}</small>
        </div>

        <button className="secondary-button" onClick={logout}>
          Logout
        </button>
      </aside>

      <section className="app-content">
        {authNotice ? (
          <div className="notice-banner" role="alert">
            <span>{authNotice}</span>
            <button className="ghost-button" onClick={clearAuthNotice}>
              Dismiss
            </button>
          </div>
        ) : null}

        <Outlet />
      </section>
    </main>
  );
}
