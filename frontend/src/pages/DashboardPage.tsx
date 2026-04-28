import { useEffect, useState } from "react";
import { apiClient } from "../api/client";
import { useAuth } from "../auth/AuthContext";

type HealthResponse = {
  success: true;
  data: {
    status: string;
  };
};

export function DashboardPage() {
  const { currentUser, refreshCurrentUser } = useAuth();
  const [healthStatus, setHealthStatus] = useState("Checking...");
  const [pageError, setPageError] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function loadHealth() {
      try {
        const response = await apiClient.requestAbsolute<HealthResponse>("/health");
        if (!cancelled) {
          setHealthStatus(response.data.status);
        }
      } catch (error) {
        if (!cancelled) {
          setHealthStatus("unreachable");
          setPageError((error as Error).message);
        }
      }
    }

    void loadHealth();

    return () => {
      cancelled = true;
    };
  }, []);

  async function handleRefresh() {
    setIsRefreshing(true);
    setPageError(null);
    try {
      await refreshCurrentUser();
    } catch (error) {
      setPageError((error as Error).message);
    } finally {
      setIsRefreshing(false);
    }
  }

  return (
    <div className="dashboard-grid">
      <section className="content-hero">
        <p className="eyebrow">Authenticated area</p>
        <h2>Frontend foundation is ready for user flows</h2>
        <p>
          Routing, token persistence, protected navigation, and centralized API
          behavior are now in place. This is the base we will build document and
          signature scenarios on top of.
        </p>
        <div className="hero-meta">
          <div>
            <span className="meta-label">Backend health</span>
            <strong>{healthStatus}</strong>
          </div>
          <div>
            <span className="meta-label">Current user</span>
            <strong>{currentUser?.email ?? "Unknown"}</strong>
          </div>
        </div>
      </section>

      {pageError ? (
        <section className="panel status-panel">
          <h3>Request issue</h3>
          <p>{pageError}</p>
        </section>
      ) : null}

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>Profile snapshot</h3>
            <p>Data loaded from the protected `/auth/me` endpoint.</p>
          </div>
          <button className="secondary-button" onClick={handleRefresh} disabled={isRefreshing}>
            {isRefreshing ? "Refreshing..." : "Refresh profile"}
          </button>
        </div>
        <pre>{JSON.stringify(currentUser, null, 2)}</pre>
      </section>

      <section className="panel empty-panel">
        <h3>Empty state placeholder</h3>
        <p>
          Documents, signatures, and audits are not mounted here yet. This panel
          intentionally exists so the app already has a designed empty state for
          the next feature slices.
        </p>
      </section>
    </div>
  );
}
