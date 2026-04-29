import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { apiBaseUrl, apiClient } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { LocaleSwitcher } from "../components/LocaleSwitcher";
import { SecurityNotice } from "../components/SecurityNotice";
import { translate } from "../locales";
import { useLocale } from "../locales/LocaleContext";

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
    return translate("validation.emailRequired");
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalized)) {
    return translate("validation.emailInvalid");
  }
  return undefined;
}

function validateName(value: string): string | undefined {
  const normalized = value.trim();
  if (!normalized) {
    return translate("validation.nameRequired");
  }
  if (normalized.length < 2) {
    return translate("validation.nameMin");
  }
  return undefined;
}

function validatePassword(value: string): string | undefined {
  if (!value) {
    return translate("validation.passwordRequired");
  }
  if (value.length < 8) {
    return translate("validation.passwordMin");
  }
  return undefined;
}

function validatePublicKey(value: string): string | undefined {
  const normalized = value.trim();
  if (!normalized) {
    return undefined;
  }
  if (!normalized.includes("BEGIN PUBLIC KEY") || !normalized.includes("END PUBLIC KEY")) {
    return translate("validation.publicKeyInvalid");
  }
  return undefined;
}

export function RegisterPage() {
  const { t } = useLocale();
  const navigate = useNavigate();
  const { accessToken, isBootstrapping } = useAuth();
  const [email, setEmail] = useState("web-user@example.com");
  const defaultNameRef = useRef(t("register.defaultName"));
  const [name, setName] = useState(defaultNameRef.current);
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

  useEffect(() => {
    const previousValue = defaultNameRef.current;
    const nextValue = t("register.defaultName");
    if (name === previousValue) {
      setName(nextValue);
    }
    defaultNameRef.current = nextValue;
  }, [name, t]);

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

      setSuccessMessage(response.data.email);
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
        <LocaleSwitcher />
        <p className="eyebrow">{t("register.eyebrow")}</p>
        <h1>{t("register.title")}</h1>
        <p>{t("register.copy")}</p>
        <div className="hero-meta auth-meta">
          <div>
            <span className="meta-label">{t("login.apiBase")}</span>
            <strong>{apiBaseUrl}</strong>
          </div>
          <div>
            <span className="meta-label">{t("register.publicKeyPem")}</span>
            <strong>{hasPublicKey ? t("register.publicKeyAttached") : t("register.publicKeyOptional")}</strong>
          </div>
        </div>
        <SecurityNotice title={t("register.securityTitle")}>
          {t("register.securityCopy")}
        </SecurityNotice>
      </section>

      <section className="auth-panel">
        <div className="auth-panel-header">
          <h2>{t("register.panelTitle")}</h2>
          <p>{t("register.panelCopy")}</p>
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
            {t("login.email")}
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
            {t("register.name")}
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
            {t("login.password")}
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
            <span className="field-hint">{t("register.passwordHint")}</span>
            {validationErrors.password ? <span className="field-error">{validationErrors.password}</span> : null}
          </label>

          <label>
            {t("register.publicKeyPem")}
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
            <span className="field-hint">{t("register.publicKeyHint")}</span>
            {validationErrors.publicKey ? <span className="field-error">{validationErrors.publicKey}</span> : null}
          </label>

          <button type="submit" disabled={isSubmitting}>
            {isSubmitting ? t("register.creatingAccount") : t("register.createAccount")}
          </button>
        </form>

        <div className="auth-actions">
          <Link className="secondary-link" to="/login">
            {t("register.alreadyHaveAccount")}
          </Link>
        </div>
      </section>
    </main>
  );
}
