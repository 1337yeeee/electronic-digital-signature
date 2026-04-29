import { FormEvent, useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { apiBaseUrl, apiClient } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { SecurityNotice } from "../components/SecurityNotice";

type RegisterResponse = {
  success: true;
  data: {
    id: string;
    email: string;
    name: string;
    public_key_pem?: string;
    created_at: string;
    updated_at: string;
  };
};

type ValidationErrors = {
  email?: string;
  name?: string;
  password?: string;
  publicKey?: string;
};

function validateEmail(value: string): string | undefined {
  const normalized = value.trim();
  if (!normalized) {
    return "Email is required.";
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalized)) {
    return "Enter a valid email address.";
  }
  return undefined;
}

function validateName(value: string): string | undefined {
  const normalized = value.trim();
  if (!normalized) {
    return "Name is required.";
  }
  if (normalized.length < 2) {
    return "Name must contain at least 2 characters.";
  }
  return undefined;
}

function validatePassword(value: string): string | undefined {
  if (!value) {
    return "Password is required.";
  }
  if (value.length < 8) {
    return "Password must contain at least 8 characters.";
  }
  return undefined;
}

function validatePublicKey(value: string): string | undefined {
  const normalized = value.trim();
  if (!normalized) {
    return undefined;
  }
  if (!normalized.includes("BEGIN PUBLIC KEY") || !normalized.includes("END PUBLIC KEY")) {
    return "Public key must be a PEM-encoded public key block.";
  }
  return undefined;
}

export function RegisterPage() {
  const navigate = useNavigate();
  const { accessToken, isBootstrapping } = useAuth();
  const [email, setEmail] = useState("web-user@example.com");
  const [name, setName] = useState("Web User");
  const [password, setPassword] = useState("secret-password");
  const [publicKey, setPublicKey] = useState("");
  const [validationErrors, setValidationErrors] = useState<ValidationErrors>({});
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (!isBootstrapping && accessToken) {
      navigate("/app", { replace: true });
    }
  }, [accessToken, isBootstrapping, navigate]);

  const hasPublicKey = useMemo(() => publicKey.trim().length > 0, [publicKey]);

  function runValidation(): ValidationErrors {
    return {
      email: validateEmail(email),
      name: validateName(name),
      password: validatePassword(password),
      publicKey: validatePublicKey(publicKey)
    };
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    const errors = runValidation();
    setValidationErrors(errors);
    setSubmitError(null);
    setSuccessMessage(null);

    if (Object.values(errors).some(Boolean)) {
      return;
    }

    setIsSubmitting(true);

    try {
      const response = await apiClient.request<RegisterResponse>("/users/register", {
        method: "POST",
        body: JSON.stringify({
          email: email.trim(),
          name: name.trim(),
          password,
          public_key_pem: publicKey.trim() || undefined
        })
      });

      setSuccessMessage(`User ${response.data.email} registered successfully.`);
      navigate("/login", {
        replace: true,
        state: {
          registered: true,
          registeredEmail: response.data.email
        }
      });
    } catch (error) {
      setSubmitError((error as Error).message);
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <main className="auth-shell auth-shell-wide">
      <section className="auth-hero">
        <p className="eyebrow">User onboarding</p>
        <h1>Create an account and bind your public key</h1>
        <p>
          Registration now happens through the web UI. The form checks the basic
          shape of the input before it ever calls the backend, then the server
          performs the final validation and uniqueness checks.
        </p>
        <div className="hero-meta auth-meta">
          <div>
            <span className="meta-label">API base</span>
            <strong>{apiBaseUrl}</strong>
          </div>
          <div>
            <span className="meta-label">Public key</span>
            <strong>{hasPublicKey ? "Will be attached" : "Optional at registration"}</strong>
          </div>
        </div>
        <SecurityNotice title="Security note">
          Paste only a public key here. Never paste a private key, seed phrase,
          or any secret signing material into the browser UI.
        </SecurityNotice>
      </section>

      <section className="auth-panel">
        <div className="auth-panel-header">
          <h2>Register</h2>
          <p>Create your identity and optionally attach an ECDSA public key now.</p>
        </div>

        {submitError ? (
          <div className="inline-error" role="alert">
            {submitError}
          </div>
        ) : null}
        {successMessage ? (
          <div className="inline-notice" role="status">
            {successMessage}
          </div>
        ) : null}

        <form onSubmit={handleSubmit} noValidate>
          <label>
            Email
            <input
              type="email"
              autoComplete="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              onBlur={() =>
                setValidationErrors((current) => ({
                  ...current,
                  email: validateEmail(email)
                }))
              }
              required
            />
            {validationErrors.email ? <span className="field-error">{validationErrors.email}</span> : null}
          </label>

          <label>
            Name
            <input
              autoComplete="name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              onBlur={() =>
                setValidationErrors((current) => ({
                  ...current,
                  name: validateName(name)
                }))
              }
              required
            />
            {validationErrors.name ? <span className="field-error">{validationErrors.name}</span> : null}
          </label>

          <label>
            Password
            <input
              type="password"
              autoComplete="new-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              onBlur={() =>
                setValidationErrors((current) => ({
                  ...current,
                  password: validatePassword(password)
                }))
              }
              required
            />
            <span className="field-hint">Use at least 8 characters.</span>
            {validationErrors.password ? <span className="field-error">{validationErrors.password}</span> : null}
          </label>

          <label>
            Public key PEM
            <textarea
              rows={8}
              value={publicKey}
              onChange={(event) => setPublicKey(event.target.value)}
              autoComplete="off"
              autoCapitalize="off"
              spellCheck={false}
              onBlur={() =>
                setValidationErrors((current) => ({
                  ...current,
                  publicKey: validatePublicKey(publicKey)
                }))
              }
              placeholder="-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----"
            />
            <span className="field-hint">
              Optional. Paste a PEM-encoded public key if you want to bind it during onboarding.
            </span>
            {validationErrors.publicKey ? <span className="field-error">{validationErrors.publicKey}</span> : null}
          </label>

          <button type="submit" disabled={isSubmitting}>
            {isSubmitting ? "Creating account..." : "Create account"}
          </button>
        </form>

        <div className="auth-actions">
          <Link className="secondary-link" to="/login">
            Already have an account? Sign in
          </Link>
        </div>
      </section>
    </main>
  );
}
