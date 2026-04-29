import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { apiClient } from "../api/client";
import { translateStatus } from "../locales";
import { useLocale } from "../locales/LocaleContext";
import { describeApiError } from "../ui/feedback";
import { useToast } from "../ui/ToastContext";

type ApiEnvelope<T> = {
  success: true;
  data: T;
};

type DocumentDetails = {
  document_id: string;
  owner_user_id: string;
  signed_by_user_id: string;
  sent_by_user_id?: string;
  owner_email: string;
  recipient_email: string;
  original_file_name: string;
  mime_type: string;
  send_status?: string;
  created_at: string;
  signed_at: string;
  sent_at?: string;
};

type DocumentAuditResponse = {
  document_id: string;
  owner_user_id: string;
  signed_by_user_id: string;
  sent_by_user_id?: string;
  owner_email: string;
  recipient_email: string;
  original_file_name: string;
  mime_type: string;
  send_status?: string;
  created_at: string;
  signed_at: string;
  sent_at?: string;
};

type SendDocumentResponse = {
  document_id: string;
  owner_user_id: string;
  signed_by_user_id: string;
  package_id?: string;
  recipient_email: string;
  send_status: string;
  sent_by_user_id?: string;
  sent_at?: string;
};

export function DocumentDetailsPage() {
  const { t, formatDateTime } = useLocale();
  const { pushToast } = useToast();
  const { id = "" } = useParams();
  const [documentDetails, setDocumentDetails] = useState<DocumentDetails | null>(null);
  const [auditDetails, setAuditDetails] = useState<DocumentAuditResponse | null>(null);
  const [pageError, setPageError] = useState<string | null>(null);
  const [auditError, setAuditError] = useState<string | null>(null);
  const [actionNotice, setActionNotice] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isResending, setIsResending] = useState(false);
  const [isLoadingAudit, setIsLoadingAudit] = useState(false);

  async function loadDocument() {
    setPageError(null);
    try {
      const response = await apiClient.request<ApiEnvelope<DocumentDetails>>(
        `/documents/${id}`
      );
      setDocumentDetails(response.data);
    } catch (error) {
      const feedback = describeApiError(error);
      setPageError(feedback.message);
      pushToast(feedback);
    } finally {
      setIsLoading(false);
    }
  }

  useEffect(() => {
    if (!id) {
      setPageError(t("validation.documentIdMissing"));
      setIsLoading(false);
      return;
    }

    void loadDocument();
  }, [id, t]);

  async function handleResend() {
    if (!documentDetails) {
      return;
    }

    setIsResending(true);
    setPageError(null);
    setActionNotice(null);
    try {
      const response = await apiClient.request<ApiEnvelope<SendDocumentResponse>>(
        `/documents/${documentDetails.document_id}/send`,
        {
          method: "POST",
          body: JSON.stringify({
            email: documentDetails.recipient_email
          })
        }
      );

      setActionNotice(
        t("documentDetails.actionNotice", {
          email: response.data.recipient_email,
          status: translateStatus(response.data.send_status)
        })
      );
      pushToast({
        title: t("documentDetails.toastResentTitle"),
        message: t("documentDetails.toastResentMessage", { email: response.data.recipient_email }),
        tone: "success"
      });
      await loadDocument();
    } catch (error) {
      const feedback = describeApiError(error);
      setPageError(feedback.message);
      pushToast(feedback);
    } finally {
      setIsResending(false);
    }
  }

  async function handleLoadAudit() {
    if (!documentDetails) {
      return;
    }

    setIsLoadingAudit(true);
    setAuditError(null);
    try {
      const response = await apiClient.request<ApiEnvelope<DocumentAuditResponse>>(
        `/documents/${documentDetails.document_id}/audit`
      );
      setAuditDetails(response.data);
      pushToast({
        title: t("documentDetails.toastAuditTitle"),
        message: t("documentDetails.toastAuditMessage"),
        tone: "info"
      });
    } catch (error) {
      const feedback = describeApiError(error);
      setAuditError(feedback.message);
      pushToast(feedback);
    } finally {
      setIsLoadingAudit(false);
    }
  }

  return (
    <div className="dashboard-grid">
      <section className="content-hero">
        <p className="eyebrow">{t("documentDetails.eyebrow")}</p>
        <h2>{t("documentDetails.title")}</h2>
        <p>{t("documentDetails.copy")}</p>
        <div className="form-actions-row top-gap">
          <Link className="secondary-link" to="/app/documents">
            {t("documentDetails.back")}
          </Link>
          <Link className="secondary-link" to="/app/documents/flow">
            {t("documentDetails.openFlow")}
          </Link>
        </div>
      </section>

      {pageError ? (
        <section className="panel status-panel">
          <h3>{t("documentDetails.loadErrorTitle")}</h3>
          <p>{pageError}</p>
        </section>
      ) : null}
      {actionNotice ? (
        <section className="panel">
          <div className="inline-notice" role="status">
            {actionNotice}
          </div>
        </section>
      ) : null}

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>{t("documentDetails.cardTitle")}</h3>
            <p>{t("documentDetails.cardCopy")}</p>
          </div>
          <div className="form-actions-row">
            <button
              type="button"
              className="secondary-button"
              onClick={handleLoadAudit}
              disabled={isLoading || isLoadingAudit || !documentDetails}
            >
              {isLoadingAudit ? t("documentDetails.loadingAudit") : t("documentDetails.viewAudit")}
            </button>
            <button
              type="button"
              onClick={handleResend}
              disabled={isLoading || isResending || !documentDetails}
            >
              {isResending ? t("documentDetails.resending") : t("common.resend")}
            </button>
          </div>
        </div>

        {isLoading ? (
          <div className="empty-panel inline-panel">{t("documentDetails.loading")}</div>
        ) : documentDetails ? (
          <div className="profile-grid">
            <article className="panel panel-soft">
              <h3>{t("documentDetails.identityTitle")}</h3>
              <dl className="details-list compact-details">
                <div>
                  <dt>{t("scenario3.documentId")}</dt>
                  <dd>{documentDetails.document_id}</dd>
                </div>
                <div>
                  <dt>{t("scenario3.originalFile")}</dt>
                  <dd>{documentDetails.original_file_name}</dd>
                </div>
                <div>
                  <dt>{t("documentDetails.mimeType")}</dt>
                  <dd>{documentDetails.mime_type}</dd>
                </div>
                <div>
                  <dt>{t("scenario3.recipientEmailField")}</dt>
                  <dd>{documentDetails.recipient_email}</dd>
                </div>
                <div>
                  <dt>{t("documentDetails.ownerEmail")}</dt>
                  <dd>{documentDetails.owner_email}</dd>
                </div>
                <div>
                  <dt>{t("scenario3.sendStatus")}</dt>
                  <dd>{translateStatus(documentDetails.send_status)}</dd>
                </div>
              </dl>
            </article>

            <article className="panel panel-soft">
              <h3>{t("documentDetails.traceTitle")}</h3>
              <dl className="details-list compact-details">
                <div>
                  <dt>{t("scenario3.ownerUserId")}</dt>
                  <dd>{documentDetails.owner_user_id}</dd>
                </div>
                <div>
                  <dt>{t("scenario3.signedByUserId")}</dt>
                  <dd>{documentDetails.signed_by_user_id}</dd>
                </div>
                <div>
                  <dt>{t("scenario3.sentByUserId")}</dt>
                  <dd>{documentDetails.sent_by_user_id || t("documentDetails.notSentYet")}</dd>
                </div>
                <div>
                  <dt>{t("documents.createdAt")}</dt>
                  <dd>{formatDateTime(documentDetails.created_at)}</dd>
                </div>
                <div>
                  <dt>{t("scenario3.signedAt")}</dt>
                  <dd>{formatDateTime(documentDetails.signed_at)}</dd>
                </div>
                <div>
                  <dt>{t("scenario3.sentAt")}</dt>
                  <dd>{formatDateTime(documentDetails.sent_at)}</dd>
                </div>
              </dl>
            </article>
          </div>
        ) : null}
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>{t("documentDetails.auditTitle")}</h3>
            <p>{t("documentDetails.auditCopy")}</p>
          </div>
        </div>

        {auditError ? (
          <div className="inline-error" role="alert">
            {auditError}
          </div>
        ) : null}

        {auditDetails ? (
          <dl className="details-list audit-grid">
            <div>
              <dt>{t("scenario3.documentId")}</dt>
              <dd>{auditDetails.document_id}</dd>
            </div>
            <div>
              <dt>{t("scenario3.ownerUserId")}</dt>
              <dd>{auditDetails.owner_user_id}</dd>
            </div>
            <div>
              <dt>{t("scenario3.signedByUserId")}</dt>
              <dd>{auditDetails.signed_by_user_id}</dd>
            </div>
            <div>
              <dt>{t("scenario3.sentByUserId")}</dt>
              <dd>{auditDetails.sent_by_user_id || t("common.notReturned")}</dd>
            </div>
            <div>
              <dt>{t("scenario3.recipientEmailField")}</dt>
              <dd>{auditDetails.recipient_email}</dd>
            </div>
            <div>
              <dt>{t("scenario3.sendStatus")}</dt>
              <dd>{auditDetails.send_status ? translateStatus(auditDetails.send_status) : t("common.notReturned")}</dd>
            </div>
            <div>
              <dt>{t("documents.createdAt")}</dt>
              <dd>{formatDateTime(auditDetails.created_at)}</dd>
            </div>
            <div>
              <dt>{t("scenario3.sentAt")}</dt>
              <dd>{formatDateTime(auditDetails.sent_at, "common.notReturned")}</dd>
            </div>
          </dl>
        ) : (
          <div className="empty-panel inline-panel">
            {t("documentDetails.auditEmpty")}
          </div>
        )}
      </section>
    </div>
  );
}
