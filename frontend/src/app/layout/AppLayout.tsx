import { NavLink, Outlet } from "react-router-dom";
import { useAuth } from "../../auth/AuthContext";
import { LocaleSwitcher } from "../../components/LocaleSwitcher";
import { useLocale } from "../../locales/LocaleContext";

export function AppLayout() {
  const { currentUser, logout, authNotice, clearAuthNotice } = useAuth();
  const { t } = useLocale();

  return (
    <main className="app-shell">
      <aside className="app-sidebar">
        <LocaleSwitcher />
        <p className="eyebrow">{t("layout.brandEyebrow")}</p>
        <h1>{t("layout.brandTitle")}</h1>
        <p className="sidebar-copy">{t("layout.sidebarCopy")}</p>

        <nav className="app-nav">
          <NavLink to="/app" end>
            {t("layout.nav.overview")}
          </NavLink>
          <NavLink to="/app/profile">
            {t("layout.nav.profile")}
          </NavLink>
          <NavLink to="/app/documents" end>
            {t("layout.nav.myDocuments")}
          </NavLink>
          <NavLink to="/app/documents/flow">
            {t("layout.nav.documentFlow")}
          </NavLink>
          <NavLink to="/app/server-signed-message">
            {t("layout.nav.serverSigned")}
          </NavLink>
          <NavLink to="/app/signatures/verify">
            {t("layout.nav.verifySignature")}
          </NavLink>
        </nav>

        <div className="user-chip">
          <span className="meta-label">{t("layout.signedInAs")}</span>
          <strong>{currentUser?.name ?? currentUser?.email ?? t("layout.unknownUser")}</strong>
          <small>{currentUser?.email}</small>
        </div>

        <button className="secondary-button" onClick={logout}>
          {t("common.logout")}
        </button>
      </aside>

      <section className="app-content">
        {authNotice ? (
          <div className="notice-banner" role="alert">
            <span>{authNotice}</span>
            <button className="ghost-button" onClick={clearAuthNotice}>
              {t("common.dismiss")}
            </button>
          </div>
        ) : null}

        <Outlet />
      </section>
    </main>
  );
}
