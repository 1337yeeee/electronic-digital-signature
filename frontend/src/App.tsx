import { FormEvent, useMemo, useState } from "react";

type ApiError = {
  success?: false;
  error?: {
    code?: string;
    message?: string;
  };
};

type LoginEnvelope = {
  success: true;
  data: {
    access_token: string;
    token_type: string;
    expires_at: string;
    user: {
      id: string;
      email: string;
      name: string;
      public_key_pem?: string;
      created_at: string;
      updated_at: string;
    };
  };
};

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

async function requestJson<T>(
  path: string,
  init?: RequestInit
): Promise<T> {
  const response = await fetch(path, init);
  const text = await response.text();
  const payload = text ? JSON.parse(text) : {};

  if (!response.ok) {
    const apiError = payload as ApiError;
    const message =
      apiError.error?.message ??
      `Request failed with status ${response.status}`;
    throw new Error(message);
  }

  return payload as T;
}

export function App() {
  const [token, setToken] = useState("");
  const [health, setHealth] = useState("Not checked");
  const [status, setStatus] = useState("Ready");
  const [registerEmail, setRegisterEmail] = useState("web-user@example.com");
  const [registerName, setRegisterName] = useState("Web User");
  const [registerPassword, setRegisterPassword] = useState("secret-password");
  const [registerPublicKey, setRegisterPublicKey] = useState("");
  const [loginEmail, setLoginEmail] = useState("web-user@example.com");
  const [loginPassword, setLoginPassword] = useState("secret-password");
  const [meJson, setMeJson] = useState("No data loaded yet.");

  const authHeader = useMemo<Record<string, string> | undefined>(
    () => (token ? { Authorization: `Bearer ${token}` } : undefined),
    [token]
  );

  async function runHealthCheck() {
    setStatus("Checking backend health...");
    try {
      const response = await requestJson<{ data: { status: string } }>(
        "/health"
      );
      setHealth(response.data.status);
      setStatus("Backend is reachable from the frontend container.");
    } catch (error) {
      setStatus((error as Error).message);
    }
  }

  async function registerUser(event: FormEvent) {
    event.preventDefault();
    setStatus("Registering user...");
    try {
      const response = await requestJson<{ data: unknown }>(
        `${apiBaseUrl}/users/register`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            email: registerEmail,
            name: registerName,
            password: registerPassword,
            public_key_pem: registerPublicKey || undefined
          })
        }
      );
      setMeJson(JSON.stringify(response.data, null, 2));
      setStatus("User registration completed.");
    } catch (error) {
      setStatus((error as Error).message);
    }
  }

  async function login(event: FormEvent) {
    event.preventDefault();
    setStatus("Logging in...");
    try {
      const response = await requestJson<LoginEnvelope>(
        `${apiBaseUrl}/auth/login`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            email: loginEmail,
            password: loginPassword
          })
        }
      );
      setToken(response.data.access_token);
      setMeJson(JSON.stringify(response.data.user, null, 2));
      setStatus("Login completed. JWT saved in memory.");
    } catch (error) {
      setStatus((error as Error).message);
    }
  }

  async function loadMe() {
    setStatus("Loading current user...");
    try {
      const response = await requestJson<{ data: unknown }>(
        `${apiBaseUrl}/auth/me`,
        {
          headers: authHeader
        }
      );
      setMeJson(JSON.stringify(response.data, null, 2));
      setStatus("Loaded current user.");
    } catch (error) {
      setStatus((error as Error).message);
    }
  }

  return (
    <main className="page-shell">
      <section className="hero">
        <p className="eyebrow">Electronic Digital Signature Lab</p>
        <h1>Containerized web shell for the backend API</h1>
        <p className="lede">
          This frontend is intentionally small but functional: it verifies that
          the browser can reach the backend, register a user, log in, and call
          authenticated endpoints from the containerized environment.
        </p>
        <div className="hero-meta">
          <div>
            <span className="meta-label">API base</span>
            <strong>{apiBaseUrl}</strong>
          </div>
          <div>
            <span className="meta-label">Backend health</span>
            <strong>{health}</strong>
          </div>
        </div>
        <button className="primary-button" onClick={runHealthCheck}>
          Check Backend Health
        </button>
      </section>

      <section className="panel-grid">
        <article className="panel">
          <h2>Register</h2>
          <p>Create a user against the live backend API.</p>
          <form onSubmit={registerUser}>
            <label>
              Email
              <input
                value={registerEmail}
                onChange={(event) => setRegisterEmail(event.target.value)}
              />
            </label>
            <label>
              Name
              <input
                value={registerName}
                onChange={(event) => setRegisterName(event.target.value)}
              />
            </label>
            <label>
              Password
              <input
                type="password"
                value={registerPassword}
                onChange={(event) => setRegisterPassword(event.target.value)}
              />
            </label>
            <label>
              Public key PEM
              <textarea
                rows={6}
                value={registerPublicKey}
                onChange={(event) => setRegisterPublicKey(event.target.value)}
                placeholder="Optional PEM-encoded ECDSA public key"
              />
            </label>
            <button type="submit">Register User</button>
          </form>
        </article>

        <article className="panel">
          <h2>Login</h2>
          <p>Obtain JWT and call protected API routes.</p>
          <form onSubmit={login}>
            <label>
              Email
              <input
                value={loginEmail}
                onChange={(event) => setLoginEmail(event.target.value)}
              />
            </label>
            <label>
              Password
              <input
                type="password"
                value={loginPassword}
                onChange={(event) => setLoginPassword(event.target.value)}
              />
            </label>
            <button type="submit">Login</button>
          </form>

          <div className="token-card">
            <span className="meta-label">In-memory access token</span>
            <code>{token ? `${token.slice(0, 48)}...` : "Not logged in"}</code>
          </div>

          <button
            className="secondary-button"
            onClick={loadMe}
            disabled={!token}
          >
            Load /auth/me
          </button>
        </article>
      </section>

      <section className="result-panel">
        <div className="result-header">
          <h2>Live response preview</h2>
          <span>{status}</span>
        </div>
        <pre>{meJson}</pre>
      </section>
    </main>
  );
}
