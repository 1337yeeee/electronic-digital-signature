import { ChangeEvent, FormEvent, useMemo, useState } from "react";
import { apiClient } from "../api/client";
import { SecurityNotice } from "../components/SecurityNotice";
import { translate, translateStatus } from "../locales";
import { useLocale } from "../locales/LocaleContext";
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
    return translate("validation.recipientRequired");
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalized)) {
    return translate("validation.recipientInvalid");
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
  const { t } = useLocale();
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
      return translate("validation.documentRequired");
    }
    if (!file.name.toLowerCase().endsWith(".docx")) {
      return translate("validation.documentExtension");
    }
    if (file.size <= 0) {
      return translate("validation.documentRequired");
    }
    if (file.size > maxDocxSizeBytes) {
      return translate("validation.documentMaxSize");
    }
    if (file.type && !allowedMimeTypes.includes(file.type)) {
      return translate("validation.documentMime");
    }
    return undefined;
  }

  function validateVerifyInput(): string | undefined {
    if (verifyInputMode === "file") {
      if (!packageFile) {
        return translate("validation.packageFileRequired");
      }
      return undefined;
    }

    if (!packageJson.trim()) {
      return translate("validation.packageJsonRequired");
    }

    try {
      JSON.parse(packageJson);
    } catch {
      return translate("validation.packageJsonInvalid");
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
        title: t("scenario3.toastUploadedTitle"),
        message: t("scenario3.toastUploadedMessage", { fileName: response.data.original_file_name }),
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
        title: t("scenario3.toastSentTitle"),
        message: t("scenario3.toastSentMessage", { email: response.data.recipient_email }),
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
      setAuditError(t("scenario3.sendEmpty"));
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
        title: t("scenario3.toastAuditTitle"),
        message: t("scenario3.toastAuditMessage"),
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
              title: t("scenario3.toastVerifiedTitle"),
              message: t("scenario3.toastVerifiedMessage"),
              tone: "success"
            }
          : {
              title: t("scenario3.toastInvalidTitle"),
              message: response.data.error || t("scenario3.toastInvalidMessage"),
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
        <p className="eyebrow">{t("scenario3.eyebrow")}</p>
        <h2>{t("scenario3.title")}</h2>
        <p>{t("scenario3.copy")}</p>
        <SecurityNotice title={t("scenario3.securityTitle")}>
          {t("scenario3.securityCopy")}
        </SecurityNotice>
      </section>

      <section className="scenario-grid">
        <article className="panel">
          <div className="panel-header">
            <div>
              <h3>{t("scenario3.uploadTitle")}</h3>
              <p>{t("scenario3.uploadCopy")}</p>
            </div>
          </div>

          {uploadError ? (
            <div className="inline-error" role="alert">
              {uploadError}
            </div>
          ) : null}

          <form onSubmit={handleUpload} noValidate>
            <label>
              {t("scenario3.recipientEmail")}
              <input
                type="email"
                value={recipientEmail}
                onChange={(event) => setRecipientEmail(event.target.value)}
                placeholder="recipient@example.com"
              />
            </label>

            <label>
              {t("scenario3.docxFile")}
              <input type="file" accept=".docx" onChange={handleUploadFileChange} />
              <span className="field-hint">{t("scenario3.docxHint")}</span>
            </label>

            <div className="form-actions-row">
              <button type="submit" disabled={isUploading}>
                {isUploading ? t("scenario3.uploadingButton") : t("scenario3.uploadButton")}
              </button>
            </div>
          </form>

          {uploadResult ? (
            <dl className="details-list top-gap">
              <div>
                <dt>{t("scenario3.documentId")}</dt>
                <dd>{uploadResult.document_id}</dd>
              </div>
              <div>
                <dt>{t("scenario3.originalFile")}</dt>
                <dd>{uploadResult.original_file_name}</dd>
              </div>
              <div>
                <dt>{t("scenario3.recipient")}</dt>
                <dd>{uploadResult.recipient_email}</dd>
              </div>
              <div>
                <dt>{t("scenario3.signedByUserId")}</dt>
                <dd>{uploadResult.signed_by_user_id}</dd>
              </div>
            </dl>
          ) : null}
        </article>

        <article className="panel">
          <div className="panel-header">
            <div>
              <h3>{t("scenario3.sendTitle")}</h3>
              <p>{t("scenario3.sendCopy")}</p>
            </div>
          </div>

          {sendError ? (
            <div className="inline-error" role="alert">
              {sendError}
            </div>
          ) : null}

          <div className="form-actions-row">
            <button type="button" disabled={!canSend || isSending} onClick={handleSend}>
              {isSending ? t("scenario3.sendingButton") : t("scenario3.sendButton")}
            </button>
            <button
              type="button"
              className="secondary-button"
              disabled={!activeDocumentId || isLoadingAudit}
              onClick={handleLoadAudit}
            >
              {isLoadingAudit ? t("scenario3.loadingAuditButton") : t("scenario3.loadAuditButton")}
            </button>
          </div>

          {sendResult ? (
            <dl className="details-list top-gap">
              <div>
                <dt>{t("scenario3.packageId")}</dt>
                <dd>{sendResult.package_id || t("common.notReturned")}</dd>
              </div>
              <div>
                <dt>{t("scenario3.status")}</dt>
                <dd>{translateStatus(sendResult.send_status)}</dd>
              </div>
              <div>
                <dt>{t("scenario3.recipient")}</dt>
                <dd>{sendResult.recipient_email}</dd>
              </div>
              <div>
                <dt>{t("scenario3.sentByUserId")}</dt>
                <dd>{sendResult.sent_by_user_id || t("common.notReturned")}</dd>
              </div>
            </dl>
          ) : (
            <div className="empty-panel inline-panel top-gap">
              {t("scenario3.sendEmpty")}
            </div>
          )}
        </article>
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>{t("scenario3.auditTitle")}</h3>
            <p>{t("scenario3.auditCopy")}</p>
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
              <dt>{t("scenario3.documentId")}</dt>
              <dd>{auditResult.document_id}</dd>
            </div>
            <div>
              <dt>{t("scenario3.ownerUserId")}</dt>
              <dd>{auditResult.owner_user_id}</dd>
            </div>
            <div>
              <dt>{t("scenario3.signedByUserId")}</dt>
              <dd>{auditResult.signed_by_user_id}</dd>
            </div>
            <div>
              <dt>{t("scenario3.sentByUserId")}</dt>
              <dd>{auditResult.sent_by_user_id || t("common.notReturned")}</dd>
            </div>
            <div>
              <dt>{t("scenario3.ownerEmail")}</dt>
              <dd>{auditResult.owner_email}</dd>
            </div>
            <div>
              <dt>{t("scenario3.recipientEmailField")}</dt>
              <dd>{auditResult.recipient_email}</dd>
            </div>
            <div>
              <dt>{t("scenario3.originalFile")}</dt>
              <dd>{auditResult.original_file_name}</dd>
            </div>
            <div>
              <dt>{t("scenario3.sendStatus")}</dt>
              <dd>{auditResult.send_status ? translateStatus(auditResult.send_status) : t("common.notReturned")}</dd>
            </div>
            <div>
              <dt>{t("scenario3.signedAt")}</dt>
              <dd>{auditResult.signed_at}</dd>
            </div>
            <div>
              <dt>{t("scenario3.sentAt")}</dt>
              <dd>{auditResult.sent_at || t("common.notReturned")}</dd>
            </div>
          </dl>
        ) : (
          <div className="empty-panel inline-panel">
            {t("scenario3.auditEmpty")}
          </div>
        )}
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>{t("scenario3.verifyTitle")}</h3>
            <p>{t("scenario3.verifyCopy")}</p>
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
            {t("scenario3.modeFile")}
          </button>
          <button
            type="button"
            className={verifyInputMode === "json" ? "secondary-button active-pill" : "secondary-button"}
            onClick={() => {
              setVerifyInputMode("json");
              setVerifyInputError(null);
            }}
          >
            {t("scenario3.modeJson")}
          </button>
        </div>

        <form onSubmit={handleVerifyPackage} noValidate>
          {verifyInputMode === "file" ? (
            <label>
              {t("scenario3.packageFile")}
              <input type="file" accept=".json,application/json" onChange={handlePackageFileChange} />
            </label>
          ) : (
            <label>
              {t("scenario3.packageJson")}
              <textarea
                rows={12}
                value={packageJson}
                onChange={(event) => setPackageJson(event.target.value)}
                autoComplete="off"
                autoCapitalize="off"
                spellCheck={false}
                placeholder={t("scenario3.packageJsonPlaceholder")}
              />
            </label>
          )}

          {verifyInputError ? <div className="field-error">{verifyInputError}</div> : null}

          <div className="form-actions-row">
            <button type="submit" disabled={isVerifying}>
              {isVerifying ? t("scenario3.verifyingButton") : t("scenario3.verifyButton")}
            </button>
            <span className="field-hint">{t("scenario3.verifyHint")}</span>
          </div>
        </form>

        {verifyResult ? (
          <div className="result-stack top-gap">
            <div className={verifyResult.valid ? "result-chip success" : "result-chip danger"}>
              {verifyResult.valid ? t("common.valid") : t("common.invalid")}
            </div>

            <dl className="details-list audit-grid">
              <div>
                <dt>{t("scenario3.documentId")}</dt>
                <dd>{verifyResult.metadata.document_id}</dd>
              </div>
              <div>
                <dt>{t("scenario3.originalFile")}</dt>
                <dd>{verifyResult.metadata.original_file_name}</dd>
              </div>
              <div>
                <dt>{t("scenario3.encryptionAlgorithm")}</dt>
                <dd>{verifyResult.metadata.encryption_algorithm}</dd>
              </div>
              <div>
                <dt>{t("scenario3.signatureAlgorithm")}</dt>
                <dd>{verifyResult.metadata.signature_algorithm}</dd>
              </div>
              <div>
                <dt>{t("scenario3.keyTransport")}</dt>
                <dd>{verifyResult.metadata.key_transport}</dd>
              </div>
              <div>
                <dt>{t("scenario3.hashBase64")}</dt>
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
                {t("scenario3.downloadDecrypted")}
              </button>
            </div>
          </div>
        ) : null}
      </section>
    </div>
  );
}
