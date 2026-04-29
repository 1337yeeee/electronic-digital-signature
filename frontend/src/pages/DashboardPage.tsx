import { useEffect, useState } from "react";
import { apiClient } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { translateStatus } from "../locales";
import { useLocale } from "../locales/LocaleContext";

type HealthResponse = {
  success: true;
  data: {
    status: string;
  };
};

export function DashboardPage() {
  const { currentUser, refreshCurrentUser } = useAuth();
  const { t } = useLocale();
  const [healthStatus, setHealthStatus] = useState(t("common.loading"));
  const [pageError, setPageError] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function loadHealth() {
      try {
        const response = await apiClient.requestAbsolute<HealthResponse>("/health");
        if (!cancelled) {
          setHealthStatus(translateStatus(response.data.status));
        }
      } catch (error) {
        if (!cancelled) {
          setHealthStatus(translateStatus("unreachable"));
          setPageError((error as Error).message);
        }
      }
    }

    void loadHealth();

    return () => {
      cancelled = true;
    };
  }, [t]);

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
        <p className="eyebrow">{t("dashboard.eyebrow")}</p>
        <h2>{t("dashboard.title")}</h2>
        <p>{t("dashboard.copy")}</p>
        <div className="hero-meta">
          <div>
            <span className="meta-label">{t("dashboard.backendHealth")}</span>
            <strong>{healthStatus}</strong>
          </div>
          <div>
            <span className="meta-label">{t("dashboard.currentUser")}</span>
            <strong>{currentUser?.email ?? t("common.unknown")}</strong>
          </div>
        </div>
      </section>

      {pageError ? (
        <section className="panel status-panel">
          <h3>{t("dashboard.requestIssue")}</h3>
          <p>{pageError}</p>
        </section>
      ) : null}

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>{t("dashboard.profileSnapshot")}</h3>
            <p>{t("dashboard.profileSnapshotCopy")}</p>
          </div>
          <button className="secondary-button" onClick={handleRefresh} disabled={isRefreshing}>
            {isRefreshing ? t("common.refreshing") : t("dashboard.refreshProfile")}
          </button>
        </div>
        <pre>{JSON.stringify(currentUser, null, 2)}</pre>
      </section>

      <section className="panel empty-panel">
        <h3>{t("dashboard.emptyTitle")}</h3>
        <p>{t("dashboard.emptyCopy")}</p>
      </section>
    </div>
  );
}
