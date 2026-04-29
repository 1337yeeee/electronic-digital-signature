import { FormEvent, useEffect, useMemo, useState } from "react";
import { ApiClientError, apiClient } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { SecurityNotice } from "../components/SecurityNotice";
import { translate } from "../locales";
import { useLocale } from "../locales/LocaleContext";
import type { User } from "../types/auth";

type UpdatePublicKeyResponse = {
  success: true;
  data: User;
};

function validatePublicKeyPem(value: string): string | undefined {
  const normalized = value.trim();

  if (!normalized) {
    return translate("validation.publicKeyRequired");
  }

  if (
    !normalized.includes("BEGIN PUBLIC KEY") ||
    !normalized.includes("END PUBLIC KEY")
  ) {
    return translate("validation.publicKeyInvalid");
  }

  return undefined;
}

export function ProfilePage() {
  const { currentUser, refreshCurrentUser } = useAuth();
  const { t, formatDateTime } = useLocale();
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
      setSuccessMessage(t("profile.updateSuccess"));
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
        <p className="eyebrow">{t("profile.eyebrow")}</p>
        <h2>{t("profile.title")}</h2>
        <p>{t("profile.copy")}</p>
        <SecurityNotice title={t("profile.securityTitle")}>
          {t("profile.securityCopy")}
        </SecurityNotice>
      </section>

      <section className="profile-grid">
        <article className="panel">
          <div className="panel-header">
            <div>
              <h3>{t("profile.identityTitle")}</h3>
              <p>{t("profile.identityCopy")}</p>
            </div>
            <button
              className="secondary-button"
              onClick={handleRefresh}
              disabled={isRefreshing}
            >
              {isRefreshing ? t("common.refreshing") : t("common.refresh")}
            </button>
          </div>

          <dl className="details-list">
            <div>
              <dt>{t("profile.userId")}</dt>
              <dd>{currentUser?.id ?? t("common.notAvailable")}</dd>
            </div>
            <div>
              <dt>{t("profile.email")}</dt>
              <dd>{currentUser?.email ?? t("common.notAvailable")}</dd>
            </div>
            <div>
              <dt>{t("profile.name")}</dt>
              <dd>{currentUser?.name ?? t("common.notAvailable")}</dd>
            </div>
            <div>
              <dt>{t("profile.createdAt")}</dt>
              <dd>{formatDateTime(currentUser?.created_at)}</dd>
            </div>
            <div>
              <dt>{t("profile.updatedAt")}</dt>
              <dd>{formatDateTime(currentUser?.updated_at)}</dd>
            </div>
          </dl>
        </article>

        <article className="panel">
          <div className="panel-header">
            <div>
              <h3>{t("profile.currentPublicKeyTitle")}</h3>
              <p>{t("profile.currentPublicKeyCopy")}</p>
            </div>
          </div>

          <pre className="pem-preview">{currentUser?.public_key_pem || t("profile.noPublicKey")}</pre>
        </article>
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>{t("profile.updateTitle")}</h3>
            <p>{t("profile.updateCopy")}</p>
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
            {t("profile.publicKeyPem")}
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
            <span className="field-hint">{t("profile.publicKeyHint")}</span>
            {pemError ? <span className="field-error">{pemError}</span> : null}
          </label>

          <div className="form-actions-row">
            <button type="submit" disabled={isSaving || !hasChanges}>
              {isSaving ? t("profile.updatingButton") : t("profile.updateButton")}
            </button>
            {!hasChanges ? (
              <span className="field-hint">{t("profile.noUnsavedChanges")}</span>
            ) : null}
          </div>
        </form>
      </section>
    </div>
  );
}
