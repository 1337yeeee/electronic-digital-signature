import { FormEvent, useEffect, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { apiBaseUrl } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { LocaleSwitcher } from "../components/LocaleSwitcher";
import { useLocale } from "../locales/LocaleContext";
import { describeApiError } from "../ui/feedback";
import { useToast } from "../ui/ToastContext";

type LocationState = {
  from?: {
    pathname?: string;
  };
  registered?: boolean;
  registeredEmail?: string;
};

export function LoginPage() {
  const { t } = useLocale();
  const { pushToast } = useToast();
  const navigate = useNavigate();
  const location = useLocation();
  const { accessToken, login, authNotice, clearAuthNotice, isBootstrapping } = useAuth();
  const locationState = location.state as LocationState | null;
  const [email, setEmail] = useState(locationState?.registeredEmail ?? "web-user@example.com");
  const [password, setPassword] = useState("secret-password");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const nextPath = locationState?.from?.pathname || "/app";

  useEffect(() => {
    if (!isBootstrapping && accessToken) {
      navigate(nextPath, { replace: true });
    }
  }, [accessToken, isBootstrapping, navigate, nextPath]);

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setIsSubmitting(true);
    setFormError(null);
    clearAuthNotice();

    try {
      await login(email, password);
      pushToast({
        title: t("login.toastTitle"),
        message: t("login.toastMessage"),
        tone: "success"
      });
      navigate(nextPath, { replace: true });
    } catch (error) {
      const feedback = describeApiError(error);
      setFormError(feedback.message);
      pushToast(feedback);
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <main className="auth-shell">
      <section className="auth-hero">
        <LocaleSwitcher />
        <p className="eyebrow">{t("login.eyebrow")}</p>
        <h1>{t("login.title")}</h1>
        <p>{t("login.copy")}</p>
        <div className="hero-meta auth-meta">
          <div>
            <span className="meta-label">{t("login.apiBase")}</span>
            <strong>{apiBaseUrl}</strong>
          </div>
          <div>
            <span className="meta-label">{t("login.flow")}</span>
            <strong>{t("login.flowValue")}</strong>
          </div>
        </div>
      </section>

      <section className="auth-panel">
        <div className="auth-panel-header">
          <h2>{t("login.panelTitle")}</h2>
          <p>{t("login.panelCopy")}</p>
        </div>

        {authNotice ? (
          <div className="inline-notice" role="alert">
            {authNotice}
          </div>
        ) : null}
        {locationState?.registered ? (
          <div className="inline-notice" role="status">
            {t("login.registrationCompleted")}
          </div>
        ) : null}
        {formError ? (
          <div className="inline-error" role="alert">
            {formError}
          </div>
        ) : null}

        <form onSubmit={handleSubmit}>
          <label>
            {t("login.email")}
            <input
              type="email"
              autoComplete="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              required
            />
          </label>
          <label>
            {t("login.password")}
            <input
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              required
            />
          </label>
          <button type="submit" disabled={isSubmitting}>
            {isSubmitting ? t("login.signingIn") : t("login.signIn")}
          </button>
        </form>

        <p className="auth-footnote">{t("login.footnote")}</p>
        <div className="auth-actions">
          <Link className="secondary-link" to="/register">
            {t("login.createAccount")}
          </Link>
          <Link className="secondary-link" to="/app">
            {t("login.tryProtectedRoute")}
          </Link>
        </div>
      </section>
    </main>
  );
}
