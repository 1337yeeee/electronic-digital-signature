import { ChangeEvent, FormEvent, useMemo, useState } from "react";
import { apiClient } from "../api/client";
import { SecurityNotice } from "../components/SecurityNotice";
import { describeApiError } from "../ui/feedback";
import { useToast } from "../ui/ToastContext";

type ApiEnvelope<T> = {
  success: true;
  data: T;
};

type UploadDocumentResponse = {
  document_id: string;
  owner_user_id: string;
  signed_by_user_id: string;
  owner_email: string;
  recipient_email: string;
  original_file_name: string;
  stored_path: string;
  mime_type: string;
  created_at: string;
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

type VerifyDecryptPackageMetadata = {
  document_id: string;
  version: string;
  encryption_algorithm: string;
  key_transport: string;
  signature_algorithm: string;
  original_file_name: string;
  mime_type: string;
  hash_base64: string;
};

type VerifyDecryptPackageResponse = {
  valid: boolean;
  error?: string;
  metadata: VerifyDecryptPackageMetadata;
  decrypted_document_base64?: string;
};

const maxDocxSizeBytes = 10 * 1024 * 1024;
const allowedMimeTypes = [
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "application/octet-stream"
];

function validateEmail(value: string): string | undefined {
  const normalized = value.trim();
  if (!normalized) {
    return "Recipient email is required.";
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalized)) {
    return "Enter a valid recipient email.";
  }
  return undefined;
}

function downloadBase64File(base64: string, fileName: string, mimeType: string) {
  const binary = atob(base64);
  const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
  const blob = new Blob([bytes], { type: mimeType || "application/octet-stream" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = fileName;
  anchor.click();
  URL.revokeObjectURL(url);
}

export function DocumentFlowPage() {
  const { pushToast } = useToast();
  const [recipientEmail, setRecipientEmail] = useState("recipient@example.com");
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [sendError, setSendError] = useState<string | null>(null);
  const [auditError, setAuditError] = useState<string | null>(null);
  const [verifyError, setVerifyError] = useState<string | null>(null);
  const [verifyInputMode, setVerifyInputMode] = useState<"file" | "json">("file");
  const [packageFile, setPackageFile] = useState<File | null>(null);
  const [packageJson, setPackageJson] = useState("");
  const [verifyInputError, setVerifyInputError] = useState<string | null>(null);
  const [uploadResult, setUploadResult] = useState<UploadDocumentResponse | null>(null);
  const [sendResult, setSendResult] = useState<SendDocumentResponse | null>(null);
  const [auditResult, setAuditResult] = useState<DocumentAuditResponse | null>(null);
  const [verifyResult, setVerifyResult] = useState<VerifyDecryptPackageResponse | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const [isLoadingAudit, setIsLoadingAudit] = useState(false);
  const [isVerifying, setIsVerifying] = useState(false);

  const activeDocumentId = sendResult?.document_id || uploadResult?.document_id || "";
  const decryptedFileName =
    verifyResult?.metadata.original_file_name || "decrypted-document.docx";
  const decryptedMimeType =
    verifyResult?.metadata.mime_type ||
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document";

  const canSend = useMemo(() => {
    return Boolean(activeDocumentId && !validateEmail(recipientEmail));
  }, [activeDocumentId, recipientEmail]);

  function validateDocxFile(file: File | null): string | undefined {
    if (!file) {
      return "Document file is required.";
    }
    if (!file.name.toLowerCase().endsWith(".docx")) {
      return "Document file must have .docx extension.";
    }
    if (file.size <= 0) {
      return "Document file is required.";
    }
    if (file.size > maxDocxSizeBytes) {
      return "Document file exceeds 10 MB.";
    }
    if (file.type && !allowedMimeTypes.includes(file.type)) {
      return "Document MIME type is not supported.";
    }
    return undefined;
  }

  function validateVerifyInput(): string | undefined {
    if (verifyInputMode === "file") {
      if (!packageFile) {
        return "Package file is required.";
      }
      return undefined;
    }

    if (!packageJson.trim()) {
      return "Package JSON is required.";
    }

    try {
      JSON.parse(packageJson);
    } catch {
      return "Package JSON must be valid JSON.";
    }

    return undefined;
  }

  function handleUploadFileChange(event: ChangeEvent<HTMLInputElement>) {
    const nextFile = event.target.files?.[0] ?? null;
    setUploadFile(nextFile);
    setUploadError(validateDocxFile(nextFile) ?? null);
  }

  function handlePackageFileChange(event: ChangeEvent<HTMLInputElement>) {
    const nextFile = event.target.files?.[0] ?? null;
    setPackageFile(nextFile);
    setVerifyInputError(null);
  }

  async function handleUpload(event: FormEvent) {
    event.preventDefault();

    const fileError = validateDocxFile(uploadFile) ?? null;
    const emailError = validateEmail(recipientEmail) ?? null;
    setUploadError(fileError || emailError);
    setSendResult(null);
    setAuditResult(null);
    setVerifyResult(null);
    setSendError(null);
    setAuditError(null);
    setVerifyError(null);

    if (fileError || emailError || !uploadFile) {
      return;
    }

    const formData = new FormData();
    formData.append("file", uploadFile);
    formData.append("recipient_email", recipientEmail.trim());

    setIsUploading(true);
    try {
      const response = await apiClient.request<ApiEnvelope<UploadDocumentResponse>>(
        "/documents",
        {
          method: "POST",
          body: formData
        }
      );
      setUploadResult(response.data);
      pushToast({
        title: "Document uploaded",
        message: `Document ${response.data.original_file_name} is ready for sending.`,
        tone: "success"
      });
    } catch (error) {
      const feedback = describeApiError(error);
      setUploadError(feedback.message);
      pushToast(feedback);
    } finally {
      setIsUploading(false);
    }
  }

  async function handleSend() {
    const emailError = validateEmail(recipientEmail) ?? null;
    setSendError(emailError);
    setAuditError(null);
    if (emailError || !activeDocumentId) {
      return;
    }

    setIsSending(true);
    try {
      const response = await apiClient.request<ApiEnvelope<SendDocumentResponse>>(
        `/documents/${activeDocumentId}/send`,
        {
          method: "POST",
          body: JSON.stringify({
            email: recipientEmail.trim()
          })
        }
      );
      setSendResult(response.data);
      pushToast({
        title: "Package sent",
        message: `Encrypted package sent to ${response.data.recipient_email}.`,
        tone: "success"
      });
    } catch (error) {
      const feedback = describeApiError(error);
      setSendError(feedback.message);
      pushToast(feedback);
    } finally {
      setIsSending(false);
    }
  }

  async function handleLoadAudit() {
    if (!activeDocumentId) {
      setAuditError("Upload a document first.");
      return;
    }

    setIsLoadingAudit(true);
    setAuditError(null);
    try {
      const response = await apiClient.request<ApiEnvelope<DocumentAuditResponse>>(
        `/documents/${activeDocumentId}/audit`
      );
      setAuditResult(response.data);
      pushToast({
        title: "Audit loaded",
        message: "The document audit trail is now visible below.",
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

  async function handleVerifyPackage(event: FormEvent) {
    event.preventDefault();

    const inputError = validateVerifyInput() ?? null;
    setVerifyInputError(inputError);
    setVerifyError(null);
    setVerifyResult(null);
    if (inputError) {
      return;
    }

    setIsVerifying(true);
    try {
      const response = verifyInputMode === "file" && packageFile
        ? await (async () => {
            const formData = new FormData();
            formData.append("package", packageFile);
            return apiClient.request<ApiEnvelope<VerifyDecryptPackageResponse>>(
              "/documents/verify-decrypt",
              {
                method: "POST",
                body: formData
              }
            );
          })()
        : await apiClient.request<ApiEnvelope<VerifyDecryptPackageResponse>>(
            "/documents/verify-decrypt",
            {
              method: "POST",
              body: packageJson.trim()
            }
          );

      setVerifyResult(response.data);
      pushToast(
        response.data.valid
          ? {
              title: "Package verified",
              message: "The encrypted package was verified and decrypted successfully.",
              tone: "success"
            }
          : {
              title: "Package invalid",
              message: response.data.error || "The package could not be verified.",
              tone: "warning"
            }
      );
    } catch (error) {
      const feedback = describeApiError(error);
      setVerifyError(feedback.message);
      pushToast(feedback);
    } finally {
      setIsVerifying(false);
    }
  }

  return (
    <div className="dashboard-grid">
      <section className="content-hero">
        <p className="eyebrow">Scenario 3</p>
        <h2>Upload, send, audit, verify and decrypt a document package</h2>
        <p>
          This workspace walks through the full document flow in the browser:
          upload a `.docx`, send the encrypted package, inspect the audit trail,
          and verify-decrypt the package returned from mail.
        </p>
        <SecurityNotice title="Security note">
          Treat uploaded package JSON and decrypted document content as
          sensitive business data. Paste only the package content you actually
          intend to verify in this browser session.
        </SecurityNotice>
      </section>

      <section className="scenario-grid">
        <article className="panel">
          <div className="panel-header">
            <div>
              <h3>1. Upload document</h3>
              <p>Protected `POST /api/v1/documents` with multipart `.docx` upload.</p>
            </div>
          </div>

          {uploadError ? (
            <div className="inline-error" role="alert">
              {uploadError}
            </div>
          ) : null}

          <form onSubmit={handleUpload} noValidate>
            <label>
              Recipient email
              <input
                type="email"
                value={recipientEmail}
                onChange={(event) => setRecipientEmail(event.target.value)}
                placeholder="recipient@example.com"
              />
            </label>

            <label>
              `.docx` file
              <input type="file" accept=".docx" onChange={handleUploadFileChange} />
              <span className="field-hint">
                Accepted: `.docx`, up to 10 MB.
              </span>
            </label>

            <div className="form-actions-row">
              <button type="submit" disabled={isUploading}>
                {isUploading ? "Uploading..." : "Upload document"}
              </button>
            </div>
          </form>

          {uploadResult ? (
            <dl className="details-list top-gap">
              <div>
                <dt>Document ID</dt>
                <dd>{uploadResult.document_id}</dd>
              </div>
              <div>
                <dt>Original file</dt>
                <dd>{uploadResult.original_file_name}</dd>
              </div>
              <div>
                <dt>Recipient</dt>
                <dd>{uploadResult.recipient_email}</dd>
              </div>
              <div>
                <dt>Signed by user ID</dt>
                <dd>{uploadResult.signed_by_user_id}</dd>
              </div>
            </dl>
          ) : null}
        </article>

        <article className="panel">
          <div className="panel-header">
            <div>
              <h3>2. Send encrypted package</h3>
              <p>Sends the encrypted package through the dev mailer.</p>
            </div>
          </div>

          {sendError ? (
            <div className="inline-error" role="alert">
              {sendError}
            </div>
          ) : null}

          <div className="form-actions-row">
            <button type="button" disabled={!canSend || isSending} onClick={handleSend}>
              {isSending ? "Sending..." : "Send package"}
            </button>
            <button
              type="button"
              className="secondary-button"
              disabled={!activeDocumentId || isLoadingAudit}
              onClick={handleLoadAudit}
            >
              {isLoadingAudit ? "Loading audit..." : "Load audit"}
            </button>
          </div>

          {sendResult ? (
            <dl className="details-list top-gap">
              <div>
                <dt>Package ID</dt>
                <dd>{sendResult.package_id || "Not returned"}</dd>
              </div>
              <div>
                <dt>Status</dt>
                <dd>{sendResult.send_status}</dd>
              </div>
              <div>
                <dt>Recipient</dt>
                <dd>{sendResult.recipient_email}</dd>
              </div>
              <div>
                <dt>Sent by user ID</dt>
                <dd>{sendResult.sent_by_user_id || "Not returned"}</dd>
              </div>
            </dl>
          ) : (
            <div className="empty-panel inline-panel top-gap">
              Upload a document first, then send the encrypted package from here.
            </div>
          )}
        </article>
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>3. Document audit</h3>
            <p>Owner-only `GET /api/v1/documents/:id/audit` response.</p>
          </div>
        </div>

        {auditError ? (
          <div className="inline-error" role="alert">
            {auditError}
          </div>
        ) : null}

        {auditResult ? (
          <dl className="details-list audit-grid">
            <div>
              <dt>Document ID</dt>
              <dd>{auditResult.document_id}</dd>
            </div>
            <div>
              <dt>Owner user ID</dt>
              <dd>{auditResult.owner_user_id}</dd>
            </div>
            <div>
              <dt>Signed by user ID</dt>
              <dd>{auditResult.signed_by_user_id}</dd>
            </div>
            <div>
              <dt>Sent by user ID</dt>
              <dd>{auditResult.sent_by_user_id || "Not returned"}</dd>
            </div>
            <div>
              <dt>Owner email</dt>
              <dd>{auditResult.owner_email}</dd>
            </div>
            <div>
              <dt>Recipient email</dt>
              <dd>{auditResult.recipient_email}</dd>
            </div>
            <div>
              <dt>Original file</dt>
              <dd>{auditResult.original_file_name}</dd>
            </div>
            <div>
              <dt>Send status</dt>
              <dd>{auditResult.send_status || "Not returned"}</dd>
            </div>
            <div>
              <dt>Signed at</dt>
              <dd>{auditResult.signed_at}</dd>
            </div>
            <div>
              <dt>Sent at</dt>
              <dd>{auditResult.sent_at || "Not returned"}</dd>
            </div>
          </dl>
        ) : (
          <div className="empty-panel inline-panel">
            Use “Load audit” after upload or send to inspect the current document trail.
          </div>
        )}
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>4. Verify and decrypt package</h3>
            <p>Public `POST /api/v1/documents/verify-decrypt` with JSON or file input.</p>
          </div>
        </div>

        {verifyError ? (
          <div className="inline-error" role="alert">
            {verifyError}
          </div>
        ) : null}

        <div className="mode-switch-row">
          <button
            type="button"
            className={verifyInputMode === "file" ? "secondary-button active-pill" : "secondary-button"}
            onClick={() => {
              setVerifyInputMode("file");
              setVerifyInputError(null);
            }}
          >
            Upload package file
          </button>
          <button
            type="button"
            className={verifyInputMode === "json" ? "secondary-button active-pill" : "secondary-button"}
            onClick={() => {
              setVerifyInputMode("json");
              setVerifyInputError(null);
            }}
          >
            Paste package JSON
          </button>
        </div>

        <form onSubmit={handleVerifyPackage} noValidate>
          {verifyInputMode === "file" ? (
            <label>
              Package file
              <input type="file" accept=".json,application/json" onChange={handlePackageFileChange} />
            </label>
          ) : (
            <label>
              Package JSON
              <textarea
                rows={12}
                value={packageJson}
                onChange={(event) => setPackageJson(event.target.value)}
                autoComplete="off"
                autoCapitalize="off"
                spellCheck={false}
                placeholder='{"metadata": {...}, "ciphertext": "..."}'
              />
            </label>
          )}

          {verifyInputError ? <div className="field-error">{verifyInputError}</div> : null}

          <div className="form-actions-row">
            <button type="submit" disabled={isVerifying}>
              {isVerifying ? "Verifying..." : "Verify and decrypt"}
            </button>
            <span className="field-hint">
              You can use a package downloaded from Mailpit/MailHog or a raw JSON package.
            </span>
          </div>
        </form>

        {verifyResult ? (
          <div className="result-stack top-gap">
            <div className={verifyResult.valid ? "result-chip success" : "result-chip danger"}>
              {verifyResult.valid ? "valid" : "invalid"}
            </div>

            <dl className="details-list audit-grid">
              <div>
                <dt>Document ID</dt>
                <dd>{verifyResult.metadata.document_id}</dd>
              </div>
              <div>
                <dt>Original file</dt>
                <dd>{verifyResult.metadata.original_file_name}</dd>
              </div>
              <div>
                <dt>Encryption algorithm</dt>
                <dd>{verifyResult.metadata.encryption_algorithm}</dd>
              </div>
              <div>
                <dt>Signature algorithm</dt>
                <dd>{verifyResult.metadata.signature_algorithm}</dd>
              </div>
              <div>
                <dt>Key transport</dt>
                <dd>{verifyResult.metadata.key_transport}</dd>
              </div>
              <div>
                <dt>Hash base64</dt>
                <dd className="long-value">{verifyResult.metadata.hash_base64}</dd>
              </div>
            </dl>

            <div className="form-actions-row">
              <button
                type="button"
                disabled={!verifyResult.decrypted_document_base64}
                onClick={() =>
                  verifyResult.decrypted_document_base64 &&
                  downloadBase64File(
                    verifyResult.decrypted_document_base64,
                    decryptedFileName,
                    decryptedMimeType
                  )
                }
              >
                Download decrypted document
              </button>
            </div>
          </div>
        ) : null}
      </section>
    </div>
  );
}
