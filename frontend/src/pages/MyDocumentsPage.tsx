import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiClient } from "../api/client";

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

function formatDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat("en-GB", {
    dateStyle: "medium",
    timeStyle: "short"
  }).format(date);
}

export function MyDocumentsPage() {
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
      setPageError((error as Error).message);
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
      setAuditError((error as Error).message);
    } finally {
      setLoadingAuditFor(null);
    }
  }

  return (
    <div className="dashboard-grid">
      <section className="content-hero">
        <p className="eyebrow">My documents</p>
        <h2>Your document workspace</h2>
        <p>
          This list shows the documents owned by the current user, their current
          delivery status, and a quick path to the audit trail or the full
          document flow screen.
        </p>
        <div className="form-actions-row top-gap">
          <button
            type="button"
            className="secondary-button"
            onClick={() => void loadDocuments("refresh")}
            disabled={isRefreshing}
          >
            {isRefreshing ? "Refreshing..." : "Refresh list"}
          </button>
          <Link className="primary-link" to="/app/documents/flow">
            New document flow
          </Link>
        </div>
      </section>

      {pageError ? (
        <section className="panel status-panel">
          <h3>Could not load documents</h3>
          <p>{pageError}</p>
        </section>
      ) : null}

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>Document list</h3>
            <p>Protected `GET /api/v1/documents/me` for the current owner.</p>
          </div>
        </div>

        {isLoading ? (
          <div className="empty-panel inline-panel">Loading your documents...</div>
        ) : documents.length === 0 ? (
          <div className="empty-panel inline-panel">
            You do not have any documents yet. Start with the document flow screen.
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
                  <span className="status-badge">{document.send_status || "created"}</span>
                </div>

                <dl className="details-list compact-details">
                  <div>
                    <dt>Recipient</dt>
                    <dd>{document.recipient_email}</dd>
                  </div>
                  <div>
                    <dt>Created at</dt>
                    <dd>{formatDate(document.created_at)}</dd>
                  </div>
                </dl>

                <div className="form-actions-row top-gap">
                  <button
                    type="button"
                    className="secondary-button"
                    onClick={() => void handleViewAudit(document.document_id)}
                    disabled={loadingAuditFor === document.document_id}
                  >
                    {loadingAuditFor === document.document_id ? "Loading audit..." : "View audit"}
                  </button>
                  <Link className="secondary-link" to={`/app/documents/${document.document_id}`}>
                    Open details
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
            <h3>Audit details</h3>
            <p>Selected document audit appears here.</p>
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
              <dt>Document ID</dt>
              <dd>{selectedAudit.document_id}</dd>
            </div>
            <div>
              <dt>Original file</dt>
              <dd>{selectedAudit.original_file_name}</dd>
            </div>
            <div>
              <dt>Recipient email</dt>
              <dd>{selectedAudit.recipient_email}</dd>
            </div>
            <div>
              <dt>Send status</dt>
              <dd>{selectedAudit.send_status || "Not returned"}</dd>
            </div>
            <div>
              <dt>Owner user ID</dt>
              <dd>{selectedAudit.owner_user_id}</dd>
            </div>
            <div>
              <dt>Signed by user ID</dt>
              <dd>{selectedAudit.signed_by_user_id}</dd>
            </div>
            <div>
              <dt>Sent by user ID</dt>
              <dd>{selectedAudit.sent_by_user_id || "Not returned"}</dd>
            </div>
            <div>
              <dt>Created at</dt>
              <dd>{formatDate(selectedAudit.created_at)}</dd>
            </div>
          </dl>
        ) : (
          <div className="empty-panel inline-panel">
            Pick any document from the list to inspect its audit trail.
          </div>
        )}
      </section>
    </div>
  );
}
