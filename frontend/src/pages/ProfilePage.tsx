import { FormEvent, useEffect, useMemo, useState } from "react";
import { ApiClientError, apiClient } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { SecurityNotice } from "../components/SecurityNotice";
import type { User } from "../types/auth";

type UpdatePublicKeyResponse = {
  success: true;
  data: User;
};

function formatDate(value?: string): string {
  if (!value) {
    return "Not available";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat("en-GB", {
    dateStyle: "medium",
    timeStyle: "short"
  }).format(date);
}

function validatePublicKeyPem(value: string): string | undefined {
  const normalized = value.trim();

  if (!normalized) {
    return "Public key PEM is required.";
  }

  if (
    !normalized.includes("BEGIN PUBLIC KEY") ||
    !normalized.includes("END PUBLIC KEY")
  ) {
    return "Public key must be a PEM-encoded public key block.";
  }

  return undefined;
}

export function ProfilePage() {
  const { currentUser, refreshCurrentUser } = useAuth();
  const [publicKey, setPublicKey] = useState(currentUser?.public_key_pem ?? "");
  const [pemError, setPemError] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);

  useEffect(() => {
    setPublicKey(currentUser?.public_key_pem ?? "");
  }, [currentUser?.public_key_pem]);

  const hasChanges = useMemo(() => {
    return publicKey.trim() !== (currentUser?.public_key_pem ?? "").trim();
  }, [currentUser?.public_key_pem, publicKey]);

  async function handleRefresh() {
    setIsRefreshing(true);
    setSubmitError(null);
    try {
      await refreshCurrentUser();
    } catch (error) {
      setSubmitError((error as Error).message);
    } finally {
      setIsRefreshing(false);
    }
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setSubmitError(null);
    setSuccessMessage(null);

    const validationMessage = validatePublicKeyPem(publicKey);
    if (validationMessage) {
      setPemError(validationMessage);
      return;
    }

    setPemError(null);
    setIsSaving(true);

    try {
      const response = await apiClient.request<UpdatePublicKeyResponse>(
        "/users/me/public-key",
        {
          method: "PUT",
          body: JSON.stringify({
            public_key_pem: publicKey.trim()
          })
        }
      );

      setPublicKey(response.data.public_key_pem ?? "");
      setSuccessMessage("Public key updated successfully.");
      await refreshCurrentUser();
    } catch (error) {
      if (error instanceof ApiClientError) {
        if (
          error.code === "invalid_public_key" ||
          error.code === "public_key_required"
        ) {
          setPemError(error.message);
          setSubmitError(null);
          setIsSaving(false);
          return;
        }
      }

      setSubmitError((error as Error).message);
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <div className="dashboard-grid">
      <section className="content-hero">
        <p className="eyebrow">Profile</p>
        <h2>Your identity and active public key</h2>
        <p>
          This page shows the current user identity from the authenticated API
          and lets you rotate the active public key without leaving the web app.
        </p>
        <SecurityNotice title="Security note">
          Update only the public key you want the server to use for verification.
          Private keys must stay outside the browser and outside this system.
        </SecurityNotice>
      </section>

      <section className="profile-grid">
        <article className="panel">
          <div className="panel-header">
            <div>
              <h3>Identity</h3>
              <p>Primary user data loaded from the protected profile endpoint.</p>
            </div>
            <button
              className="secondary-button"
              onClick={handleRefresh}
              disabled={isRefreshing}
            >
              {isRefreshing ? "Refreshing..." : "Refresh"}
            </button>
          </div>

          <dl className="details-list">
            <div>
              <dt>User ID</dt>
              <dd>{currentUser?.id ?? "Not available"}</dd>
            </div>
            <div>
              <dt>Email</dt>
              <dd>{currentUser?.email ?? "Not available"}</dd>
            </div>
            <div>
              <dt>Name</dt>
              <dd>{currentUser?.name ?? "Not available"}</dd>
            </div>
            <div>
              <dt>Created at</dt>
              <dd>{formatDate(currentUser?.created_at)}</dd>
            </div>
            <div>
              <dt>Updated at</dt>
              <dd>{formatDate(currentUser?.updated_at)}</dd>
            </div>
          </dl>
        </article>

        <article className="panel">
          <div className="panel-header">
            <div>
              <h3>Current public key</h3>
              <p>
                The backend uses this key for current-user signature scenarios.
              </p>
            </div>
          </div>

          <pre className="pem-preview">{currentUser?.public_key_pem || "No public key is attached yet."}</pre>
        </article>
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>Update public key</h3>
            <p>Paste a PEM-encoded public key to replace the current active key.</p>
          </div>
        </div>

        {successMessage ? (
          <div className="inline-notice" role="status">
            {successMessage}
          </div>
        ) : null}
        {submitError ? (
          <div className="inline-error" role="alert">
            {submitError}
          </div>
        ) : null}

        <form onSubmit={handleSubmit} noValidate>
          <label>
            Public key PEM
            <textarea
              rows={10}
              value={publicKey}
              onChange={(event) => setPublicKey(event.target.value)}
              autoComplete="off"
              autoCapitalize="off"
              spellCheck={false}
              onBlur={() => setPemError(validatePublicKeyPem(publicKey) ?? null)}
              placeholder="-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----"
            />
            <span className="field-hint">
              Only a PEM-encoded public key is accepted. The server will also
              validate that it is a supported key type.
            </span>
            {pemError ? <span className="field-error">{pemError}</span> : null}
          </label>

          <div className="form-actions-row">
            <button type="submit" disabled={isSaving || !hasChanges}>
              {isSaving ? "Updating key..." : "Update public key"}
            </button>
            {!hasChanges ? (
              <span className="field-hint">No unsaved changes.</span>
            ) : null}
          </div>
        </form>
      </section>
    </div>
  );
}
