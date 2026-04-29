import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { apiClient } from "../api/client";
import { translateSignerType } from "../locales";
import { useLocale } from "../locales/LocaleContext";
import { describeApiError } from "../ui/feedback";
import { useToast } from "../ui/ToastContext";

type ServerPublicKeyResponse = {
  algorithm: string;
  public_key_pem: string;
};

type IssueServerMessageResponse = {
  message_id: string;
  signer_type: string;
  signer_user_id?: string;
  created_by_user_id?: string;
  created_at: string;
  message: string;
  algorithm: string;
  hash_base64: string;
  signature_base64: string;
};

function downloadText(filename: string, content: string) {
  const blob = new Blob([content], { type: "text/plain;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

async function copyToClipboard(text: string) {
  await navigator.clipboard.writeText(text);
}

function encodeUtf8Base64(value: string) {
  const bytes = new TextEncoder().encode(value);
  let binary = "";
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte);
  });
  return btoa(binary);
}

export function ServerSignedMessagePage() {
  const { t } = useLocale();
  const defaultMessageRef = useRef(t("scenario2.defaultMessage"));
  const [serverPublicKey, setServerPublicKey] = useState<ServerPublicKeyResponse | null>(null);
  const [publicKeyError, setPublicKeyError] = useState<string | null>(null);
  const [messageValue, setMessageValue] = useState(defaultMessageRef.current);
  const [requestError, setRequestError] = useState<string | null>(null);
  const [result, setResult] = useState<IssueServerMessageResponse | null>(null);
  const [isLoadingKey, setIsLoadingKey] = useState(false);
  const [isIssuing, setIsIssuing] = useState(false);
  const [copyNotice, setCopyNotice] = useState<string | null>(null);
  const { pushToast } = useToast();

  useEffect(() => {
    const nextValue = t("scenario2.defaultMessage");
    const previousValue = defaultMessageRef.current;
    if (messageValue === previousValue) {
      setMessageValue(nextValue);
    }
    defaultMessageRef.current = nextValue;
  }, [messageValue, t]);

  const exportPayload = useMemo(() => {
    if (!result && !serverPublicKey) {
      return null;
    }

    return JSON.stringify(
      {
        server_public_key: serverPublicKey?.public_key_pem ?? null,
        server_public_key_algorithm: serverPublicKey?.algorithm ?? null,
        signed_message: result
      },
      null,
      2
    );
  }, [result, serverPublicKey]);

  const verificationScript = useMemo(() => {
    if (!result || !serverPublicKey?.public_key_pem) {
      return null;
    }

    const messageBase64 = encodeUtf8Base64(result.message);

    return `#!/usr/bin/env bash
set -euo pipefail

cat > server_public.pem <<'PEM'
${serverPublicKey.public_key_pem}
PEM

cat > message.base64 <<'B64'
${messageBase64}
B64

cat > signature.base64 <<'B64'
${result.signature_base64}
B64

openssl base64 -d -A -in message.base64 -out message.txt
openssl base64 -d -A -in signature.base64 -out signature.bin

openssl dgst -sha256 -verify server_public.pem -signature signature.bin message.txt
`;
  }, [result, serverPublicKey]);

  async function loadServerPublicKey() {
    setIsLoadingKey(true);
    setPublicKeyError(null);
    setCopyNotice(null);

    try {
      const response = await apiClient.request<ServerPublicKeyResponse>(
        "/server/public-key"
      );
      setServerPublicKey(response);
    } catch (error) {
      const feedback = describeApiError(error);
      setPublicKeyError(feedback.message);
      pushToast(feedback);
    } finally {
      setIsLoadingKey(false);
    }
  }

  async function handleIssueMessage(event: FormEvent) {
    event.preventDefault();
    setRequestError(null);
    setCopyNotice(null);
    setIsIssuing(true);

    try {
      const response = await apiClient.request<IssueServerMessageResponse>(
        "/server/messages",
        {
          method: "POST",
          body: JSON.stringify({
            message: messageValue.trim() || undefined
          })
        }
      );
      setResult(response);
    } catch (error) {
      const feedback = describeApiError(error);
      setRequestError(feedback.message);
      pushToast(feedback);
    } finally {
      setIsIssuing(false);
    }
  }

  async function handleCopy(label: string, value: string) {
    try {
      await copyToClipboard(value);
      setCopyNotice(t("scenario2.copySuccess", { label }));
    } catch {
      setCopyNotice(t("scenario2.copyFailed", { label: label.toLowerCase() }));
    }
  }

  function handleExport() {
    if (!exportPayload) {
      return;
    }
    downloadText("server-signed-message.json", exportPayload);
  }

  return (
    <div className="dashboard-grid">
      <section className="content-hero">
        <p className="eyebrow">{t("scenario2.eyebrow")}</p>
        <h2>{t("scenario2.title")}</h2>
        <p>{t("scenario2.copy")}</p>
      </section>

      <section className="scenario-grid">
        <article className="panel">
          <div className="panel-header">
            <div>
              <h3>{t("scenario2.step1Title")}</h3>
              <p>{t("scenario2.step1Copy")}</p>
            </div>
            <button className="secondary-button" onClick={loadServerPublicKey} disabled={isLoadingKey}>
              {isLoadingKey ? t("scenario2.loadingKey") : t("scenario2.getPublicKey")}
            </button>
          </div>

          {publicKeyError ? (
            <div className="inline-error" role="alert">
              {publicKeyError}
            </div>
          ) : null}

          <pre className="pem-preview">
            {serverPublicKey?.public_key_pem || t("scenario2.publicKeyNotLoaded")}
          </pre>
          <p className="field-hint">
            {t("scenario2.algorithm")}: {serverPublicKey?.algorithm ?? t("scenario2.algorithmNotLoaded")}
          </p>
        </article>

        <article className="panel">
          <div className="panel-header">
            <div>
              <h3>{t("scenario2.step2Title")}</h3>
              <p>{t("scenario2.step2Copy")}</p>
            </div>
          </div>

          {requestError ? (
            <div className="inline-error" role="alert">
              {requestError}
            </div>
          ) : null}
          {copyNotice ? (
            <div className="inline-notice" role="status">
              {copyNotice}
            </div>
          ) : null}

          <form onSubmit={handleIssueMessage}>
            <label>
              {t("scenario2.message")}
              <textarea
                rows={5}
                value={messageValue}
                onChange={(event) => setMessageValue(event.target.value)}
                placeholder={t("scenario2.messagePlaceholder")}
              />
              <span className="field-hint">{t("scenario2.messageHint")}</span>
            </label>

            <div className="form-actions-row">
              <button type="submit" disabled={isIssuing}>
                {isIssuing ? t("scenario2.requestingMessage") : t("scenario2.requestMessage")}
              </button>
              <button
                type="button"
                className="secondary-button"
                onClick={handleExport}
                disabled={!exportPayload}
              >
                {t("scenario2.exportJson")}
              </button>
            </div>
          </form>
        </article>
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>{t("scenario2.payloadTitle")}</h3>
            <p>{t("scenario2.payloadCopy")}</p>
          </div>
        </div>

        {result ? (
          <div className="result-stack">
            <div className="copy-actions">
              <button
                className="secondary-button"
                type="button"
                onClick={() => handleCopy(t("scenario2.labelMessage"), result.message)}
              >
                {t("scenario2.copyMessage")}
              </button>
              <button
                className="secondary-button"
                type="button"
                onClick={() => handleCopy(t("scenario2.labelSignature"), result.signature_base64)}
              >
                {t("scenario2.copySignature")}
              </button>
              <button
                className="secondary-button"
                type="button"
                onClick={() => handleCopy(t("scenario2.labelHash"), result.hash_base64)}
              >
                {t("scenario2.copyHash")}
              </button>
              <button
                className="secondary-button"
                type="button"
                onClick={() =>
                  handleCopy(
                    t("scenario2.labelPublicKey"),
                    serverPublicKey?.public_key_pem ?? ""
                  )
                }
                disabled={!serverPublicKey?.public_key_pem}
              >
                {t("scenario2.copyPublicKey")}
              </button>
              <button
                className="secondary-button"
                type="button"
                onClick={() =>
                  verificationScript &&
                  handleCopy(t("scenario2.labelVerificationScript"), verificationScript)
                }
                disabled={!verificationScript}
              >
                {t("scenario2.copyScript")}
              </button>
            </div>

            <dl className="details-list">
              <div>
                <dt>{t("scenario2.messageField")}</dt>
                <dd>{result.message}</dd>
              </div>
              <div>
                <dt>{t("scenario2.signatureField")}</dt>
                <dd className="long-value">{result.signature_base64}</dd>
              </div>
              <div>
                <dt>{t("scenario2.hashField")}</dt>
                <dd className="long-value">{result.hash_base64}</dd>
              </div>
              <div>
                <dt>{t("scenario2.signerType")}</dt>
                <dd>{translateSignerType(result.signer_type)}</dd>
              </div>
              <div>
                <dt>{t("scenario2.createdBy")}</dt>
                <dd>{result.created_by_user_id ?? t("common.notReturned")}</dd>
              </div>
              <div>
                <dt>{t("scenario2.createdAt")}</dt>
                <dd>{result.created_at}</dd>
              </div>
              <div>
                <dt>{t("scenario2.messageId")}</dt>
                <dd>{result.message_id}</dd>
              </div>
              <div>
                <dt>{t("scenario2.algorithm")}</dt>
                <dd>{result.algorithm}</dd>
              </div>
            </dl>
          </div>
        ) : (
          <div className="empty-panel inline-panel">
            {t("scenario2.payloadEmpty")}
          </div>
        )}
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>{t("scenario2.scriptTitle")}</h3>
            <p>{t("scenario2.scriptCopy")}</p>
          </div>
        </div>

        {verificationScript ? (
          <div className="result-stack">
            <pre className="code-block">{verificationScript}</pre>
            <div className="form-actions-row">
              <button
                type="button"
                className="secondary-button"
                onClick={() => handleCopy(t("scenario2.labelVerificationScript"), verificationScript)}
              >
                {t("scenario2.copyScript")}
              </button>
            </div>
          </div>
        ) : (
          <div className="empty-panel inline-panel">
            {t("scenario2.scriptEmpty")}
          </div>
        )}
      </section>
    </div>
  );
}
