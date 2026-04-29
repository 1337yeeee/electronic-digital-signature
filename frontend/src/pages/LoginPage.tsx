import { FormEvent, useEffect, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { apiBaseUrl } from "../api/client";
import { useAuth } from "../auth/AuthContext";
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
  const { pushToast } = useToast();
  const navigate = useNavigate();
  const location = useLocation();
  const { accessToken, login, authNotice, clearAuthNotice, isBootstrapping } = useAuth();
  const locationState = location.state as LocationState | null;
  const [email, setEmail] = useState(
    locationState?.registeredEmail ?? "web-user@example.com"
  );
  const [password, setPassword] = useState("secret-password");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const nextPath =
    (locationState?.from?.pathname || "/app");

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
        title: "Signed in",
        message: "Welcome back. Your workspace is ready.",
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
        <p className="eyebrow">Scenario-ready frontend</p>
        <h1>Sign in to the digital signature workspace</h1>
        <p>
          The frontend now uses routed pages, stored JWT auth, protected routes,
          and a shared API client with centralized 401 and 403 handling.
        </p>
        <div className="hero-meta auth-meta">
          <div>
            <span className="meta-label">API base</span>
            <strong>{apiBaseUrl}</strong>
          </div>
          <div>
            <span className="meta-label">Flow</span>
            <strong>Login - JWT - Protected routes</strong>
          </div>
        </div>
      </section>

      <section className="auth-panel">
        <div className="auth-panel-header">
          <h2>Login</h2>
          <p>Use an existing registered user to enter the protected area.</p>
        </div>

        {authNotice ? (
          <div className="inline-notice" role="alert">
            {authNotice}
          </div>
        ) : null}
        {locationState?.registered ? (
          <div className="inline-notice" role="status">
            Registration completed. You can sign in now.
          </div>
        ) : null}
        {formError ? (
          <div className="inline-error" role="alert">
            {formError}
          </div>
        ) : null}

        <form onSubmit={handleSubmit}>
          <label>
            Email
            <input
              type="email"
              autoComplete="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              required
            />
          </label>
          <label>
            Password
            <input
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              required
            />
          </label>
          <button type="submit" disabled={isSubmitting}>
            {isSubmitting ? "Signing in..." : "Sign in"}
          </button>
        </form>

        <p className="auth-footnote">
          Need a user first? Create one directly in the web app and then return
          here to continue the authenticated scenarios.
        </p>
        <div className="auth-actions">
          <Link className="secondary-link" to="/register">
            Create account
          </Link>
          <Link className="secondary-link" to="/app">
            Try protected route
          </Link>
        </div>
      </section>
    </main>
  );
}
