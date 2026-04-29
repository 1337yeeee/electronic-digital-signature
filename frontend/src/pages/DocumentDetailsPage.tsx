import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { apiClient } from "../api/client";

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

export function DocumentDetailsPage() {
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
      setPageError((error as Error).message);
    } finally {
      setIsLoading(false);
    }
  }

  useEffect(() => {
    if (!id) {
      setPageError("Document id is missing.");
      setIsLoading(false);
      return;
    }

    void loadDocument();
  }, [id]);

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
        `Package resent to ${response.data.recipient_email}. Status: ${response.data.send_status}.`
      );
      await loadDocument();
    } catch (error) {
      setPageError((error as Error).message);
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
    } catch (error) {
      setAuditError((error as Error).message);
    } finally {
      setIsLoadingAudit(false);
    }
  }

  return (
    <div className="dashboard-grid">
      <section className="content-hero">
        <p className="eyebrow">Document details</p>
        <h2>Single document workspace</h2>
        <p>
          This page gives one document its own clear card: who owns it, who
          signed and sent it, when it moved through the flow, and quick actions
          to resend or inspect audit details.
        </p>
        <div className="form-actions-row top-gap">
          <Link className="secondary-link" to="/app/documents">
            Back to my documents
          </Link>
          <Link className="secondary-link" to="/app/documents/flow">
            Open document flow
          </Link>
        </div>
      </section>

      {pageError ? (
        <section className="panel status-panel">
          <h3>Could not load document</h3>
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
            <h3>Document card</h3>
            <p>Protected `GET /api/v1/documents/:id` for the current owner.</p>
          </div>
          <div className="form-actions-row">
            <button
              type="button"
              className="secondary-button"
              onClick={handleLoadAudit}
              disabled={isLoading || isLoadingAudit || !documentDetails}
            >
              {isLoadingAudit ? "Loading audit..." : "View audit"}
            </button>
            <button
              type="button"
              onClick={handleResend}
              disabled={isLoading || isResending || !documentDetails}
            >
              {isResending ? "Resending..." : "Resend"}
            </button>
          </div>
        </div>

        {isLoading ? (
          <div className="empty-panel inline-panel">Loading document details...</div>
        ) : documentDetails ? (
          <div className="profile-grid">
            <article className="panel panel-soft">
              <h3>Identity</h3>
              <dl className="details-list compact-details">
                <div>
                  <dt>Document ID</dt>
                  <dd>{documentDetails.document_id}</dd>
                </div>
                <div>
                  <dt>Original file</dt>
                  <dd>{documentDetails.original_file_name}</dd>
                </div>
                <div>
                  <dt>MIME type</dt>
                  <dd>{documentDetails.mime_type}</dd>
                </div>
                <div>
                  <dt>Recipient email</dt>
                  <dd>{documentDetails.recipient_email}</dd>
                </div>
                <div>
                  <dt>Owner email</dt>
                  <dd>{documentDetails.owner_email}</dd>
                </div>
                <div>
                  <dt>Send status</dt>
                  <dd>{documentDetails.send_status || "created"}</dd>
                </div>
              </dl>
            </article>

            <article className="panel panel-soft">
              <h3>Trace</h3>
              <dl className="details-list compact-details">
                <div>
                  <dt>Owner user ID</dt>
                  <dd>{documentDetails.owner_user_id}</dd>
                </div>
                <div>
                  <dt>Signed by user ID</dt>
                  <dd>{documentDetails.signed_by_user_id}</dd>
                </div>
                <div>
                  <dt>Sent by user ID</dt>
                  <dd>{documentDetails.sent_by_user_id || "Not sent yet"}</dd>
                </div>
                <div>
                  <dt>Created at</dt>
                  <dd>{formatDate(documentDetails.created_at)}</dd>
                </div>
                <div>
                  <dt>Signed at</dt>
                  <dd>{formatDate(documentDetails.signed_at)}</dd>
                </div>
                <div>
                  <dt>Sent at</dt>
                  <dd>{formatDate(documentDetails.sent_at)}</dd>
                </div>
              </dl>
            </article>
          </div>
        ) : null}
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>Audit</h3>
            <p>Load the audit trail for deeper event context.</p>
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
              <dt>Document ID</dt>
              <dd>{auditDetails.document_id}</dd>
            </div>
            <div>
              <dt>Owner user ID</dt>
              <dd>{auditDetails.owner_user_id}</dd>
            </div>
            <div>
              <dt>Signed by user ID</dt>
              <dd>{auditDetails.signed_by_user_id}</dd>
            </div>
            <div>
              <dt>Sent by user ID</dt>
              <dd>{auditDetails.sent_by_user_id || "Not returned"}</dd>
            </div>
            <div>
              <dt>Recipient email</dt>
              <dd>{auditDetails.recipient_email}</dd>
            </div>
            <div>
              <dt>Send status</dt>
              <dd>{auditDetails.send_status || "Not returned"}</dd>
            </div>
            <div>
              <dt>Created at</dt>
              <dd>{formatDate(auditDetails.created_at)}</dd>
            </div>
            <div>
              <dt>Sent at</dt>
              <dd>{formatDate(auditDetails.sent_at)}</dd>
            </div>
          </dl>
        ) : (
          <div className="empty-panel inline-panel">
            Use “View audit” to load the document audit trail.
          </div>
        )}
      </section>
    </div>
  );
}
