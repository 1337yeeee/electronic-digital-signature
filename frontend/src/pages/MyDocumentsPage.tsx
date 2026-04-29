import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiClient } from "../api/client";
import { translateStatus } from "../locales";
import { useLocale } from "../locales/LocaleContext";
import { describeApiError } from "../ui/feedback";
import { useToast } from "../ui/ToastContext";

type ApiEnvelope<T> = {
  success: true;
  data: T;
};

type UserDocumentListItem = {
  document_id: string;
  original_file_name: string;
  recipient_email: string;
  send_status?: string;
  created_at: string;
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

export function MyDocumentsPage() {
  const { t, formatDateTime } = useLocale();
  const { pushToast } = useToast();
  const [documents, setDocuments] = useState<UserDocumentListItem[]>([]);
  const [selectedAudit, setSelectedAudit] = useState<DocumentAuditResponse | null>(null);
  const [pageError, setPageError] = useState<string | null>(null);
  const [auditError, setAuditError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [loadingAuditFor, setLoadingAuditFor] = useState<string | null>(null);

  async function loadDocuments(mode: "initial" | "refresh" = "initial") {
    if (mode === "initial") {
      setIsLoading(true);
    } else {
      setIsRefreshing(true);
    }

    setPageError(null);

    try {
      const response = await apiClient.request<ApiEnvelope<UserDocumentListItem[]>>(
        "/documents/me"
      );
      setDocuments(response.data);
    } catch (error) {
      const feedback = describeApiError(error);
      setPageError(feedback.message);
      pushToast(feedback);
    } finally {
      if (mode === "initial") {
        setIsLoading(false);
      } else {
        setIsRefreshing(false);
      }
    }
  }

  useEffect(() => {
    void loadDocuments();
  }, []);

  async function handleViewAudit(documentID: string) {
    setLoadingAuditFor(documentID);
    setAuditError(null);

    try {
      const response = await apiClient.request<ApiEnvelope<DocumentAuditResponse>>(
        `/documents/${documentID}/audit`
      );
      setSelectedAudit(response.data);
    } catch (error) {
      const feedback = describeApiError(error);
      setAuditError(feedback.message);
      pushToast(feedback);
    } finally {
      setLoadingAuditFor(null);
    }
  }

  return (
    <div className="dashboard-grid">
      <section className="content-hero">
        <p className="eyebrow">{t("documents.eyebrow")}</p>
        <h2>{t("documents.title")}</h2>
        <p>{t("documents.copy")}</p>
        <div className="form-actions-row top-gap">
          <button
            type="button"
            className="secondary-button"
            onClick={() => void loadDocuments("refresh")}
            disabled={isRefreshing}
          >
            {isRefreshing ? t("common.refreshing") : t("documents.refreshList")}
          </button>
          <Link className="primary-link" to="/app/documents/flow">
            {t("documents.newFlow")}
          </Link>
        </div>
      </section>

      {pageError ? (
        <section className="panel status-panel">
          <h3>{t("documents.loadErrorTitle")}</h3>
          <p>{pageError}</p>
        </section>
      ) : null}

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>{t("documents.listTitle")}</h3>
            <p>{t("documents.listCopy")}</p>
          </div>
        </div>

        {isLoading ? (
          <div className="empty-panel inline-panel">{t("documents.loading")}</div>
        ) : documents.length === 0 ? (
          <div className="empty-panel inline-panel">
            {t("documents.empty")}
          </div>
        ) : (
          <div className="documents-grid">
            {documents.map((document) => (
              <article className="document-card" key={document.document_id}>
                <div className="document-card-header">
                  <div>
                    <h4>{document.original_file_name}</h4>
                    <p>{document.document_id}</p>
                  </div>
                  <span className="status-badge">{translateStatus(document.send_status)}</span>
                </div>

                <dl className="details-list compact-details">
                  <div>
                    <dt>{t("documents.recipient")}</dt>
                    <dd>{document.recipient_email}</dd>
                  </div>
                  <div>
                    <dt>{t("documents.createdAt")}</dt>
                    <dd>{formatDateTime(document.created_at)}</dd>
                  </div>
                </dl>

                <div className="form-actions-row top-gap">
                  <button
                    type="button"
                    className="secondary-button"
                    onClick={() => void handleViewAudit(document.document_id)}
                    disabled={loadingAuditFor === document.document_id}
                  >
                    {loadingAuditFor === document.document_id ? t("documents.loadingAudit") : t("documents.viewAudit")}
                  </button>
                  <Link className="secondary-link" to={`/app/documents/${document.document_id}`}>
                    {t("documents.openDetails")}
                  </Link>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>{t("documents.auditTitle")}</h3>
            <p>{t("documents.auditCopy")}</p>
          </div>
        </div>

        {auditError ? (
          <div className="inline-error" role="alert">
            {auditError}
          </div>
        ) : null}

        {selectedAudit ? (
          <dl className="details-list audit-grid">
            <div>
              <dt>{t("scenario3.documentId")}</dt>
              <dd>{selectedAudit.document_id}</dd>
            </div>
            <div>
              <dt>{t("scenario3.originalFile")}</dt>
              <dd>{selectedAudit.original_file_name}</dd>
            </div>
            <div>
              <dt>{t("scenario3.recipientEmailField")}</dt>
              <dd>{selectedAudit.recipient_email}</dd>
            </div>
            <div>
              <dt>{t("scenario3.sendStatus")}</dt>
              <dd>{selectedAudit.send_status ? translateStatus(selectedAudit.send_status) : t("common.notReturned")}</dd>
            </div>
            <div>
              <dt>{t("scenario3.ownerUserId")}</dt>
              <dd>{selectedAudit.owner_user_id}</dd>
            </div>
            <div>
              <dt>{t("scenario3.signedByUserId")}</dt>
              <dd>{selectedAudit.signed_by_user_id}</dd>
            </div>
            <div>
              <dt>{t("scenario3.sentByUserId")}</dt>
              <dd>{selectedAudit.sent_by_user_id || t("common.notReturned")}</dd>
            </div>
            <div>
              <dt>{t("documents.createdAt")}</dt>
              <dd>{formatDateTime(selectedAudit.created_at)}</dd>
            </div>
          </dl>
        ) : (
          <div className="empty-panel inline-panel">
            {t("documents.auditEmpty")}
          </div>
        )}
      </section>
    </div>
  );
}
