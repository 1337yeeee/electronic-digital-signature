import { FormEvent, useMemo, useState } from "react";
import { apiClient } from "../api/client";

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

export function ServerSignedMessagePage() {
  const [serverPublicKey, setServerPublicKey] = useState<ServerPublicKeyResponse | null>(null);
  const [publicKeyError, setPublicKeyError] = useState<string | null>(null);
  const [messageValue, setMessageValue] = useState(
    "Server, please sign this message so I can verify it externally."
  );
  const [requestError, setRequestError] = useState<string | null>(null);
  const [result, setResult] = useState<IssueServerMessageResponse | null>(null);
  const [isLoadingKey, setIsLoadingKey] = useState(false);
  const [isIssuing, setIsIssuing] = useState(false);
  const [copyNotice, setCopyNotice] = useState<string | null>(null);

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
      setPublicKeyError((error as Error).message);
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
      setRequestError((error as Error).message);
    } finally {
      setIsIssuing(false);
    }
  }

  async function handleCopy(label: string, value: string) {
    try {
      await copyToClipboard(value);
      setCopyNotice(`${label} copied to clipboard.`);
    } catch {
      setCopyNotice(`Could not copy ${label.toLowerCase()} automatically.`);
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
        <p className="eyebrow">Scenario 2</p>
        <h2>Server signs, client verifies</h2>
        <p>
          This screen lets you fetch the server public key, ask the backend to
          sign a message, and export the exact verification data for an external
          tool such as OpenSSL or a local script.
        </p>
      </section>

      <section className="scenario-grid">
        <article className="panel">
          <div className="panel-header">
            <div>
              <h3>1. Get server public key</h3>
              <p>Calls `GET /api/v1/server/public-key`.</p>
            </div>
            <button className="secondary-button" onClick={loadServerPublicKey} disabled={isLoadingKey}>
              {isLoadingKey ? "Loading..." : "Get public key"}
            </button>
          </div>

          {publicKeyError ? (
            <div className="inline-error" role="alert">
              {publicKeyError}
            </div>
          ) : null}

          <pre className="pem-preview">
            {serverPublicKey?.public_key_pem || "Server public key has not been loaded yet."}
          </pre>
          <p className="field-hint">
            Algorithm: {serverPublicKey?.algorithm ?? "Not loaded yet"}
          </p>
        </article>

        <article className="panel">
          <div className="panel-header">
            <div>
              <h3>2. Request signed message</h3>
              <p>Calls protected `POST /api/v1/server/messages`.</p>
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
              Message
              <textarea
                rows={5}
                value={messageValue}
                onChange={(event) => setMessageValue(event.target.value)}
                placeholder="Leave blank to let the server generate a message"
              />
              <span className="field-hint">
                If you clear the field entirely, the backend may generate a timestamped message for you.
              </span>
            </label>

            <div className="form-actions-row">
              <button type="submit" disabled={isIssuing}>
                {isIssuing ? "Requesting..." : "Request signed message"}
              </button>
              <button
                type="button"
                className="secondary-button"
                onClick={handleExport}
                disabled={!exportPayload}
              >
                Export JSON
              </button>
            </div>
          </form>
        </article>
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>Verification payload</h3>
            <p>Use these exact values in any external verification flow.</p>
          </div>
        </div>

        {result ? (
          <div className="result-stack">
            <div className="copy-actions">
              <button
                className="secondary-button"
                type="button"
                onClick={() => handleCopy("Message", result.message)}
              >
                Copy message
              </button>
              <button
                className="secondary-button"
                type="button"
                onClick={() => handleCopy("Signature", result.signature_base64)}
              >
                Copy signature
              </button>
              <button
                className="secondary-button"
                type="button"
                onClick={() => handleCopy("Hash", result.hash_base64)}
              >
                Copy hash
              </button>
              <button
                className="secondary-button"
                type="button"
                onClick={() =>
                  handleCopy(
                    "Public key",
                    serverPublicKey?.public_key_pem ?? ""
                  )
                }
                disabled={!serverPublicKey?.public_key_pem}
              >
                Copy public key
              </button>
            </div>

            <dl className="details-list">
              <div>
                <dt>Message</dt>
                <dd>{result.message}</dd>
              </div>
              <div>
                <dt>Signature base64</dt>
                <dd className="long-value">{result.signature_base64}</dd>
              </div>
              <div>
                <dt>Hash base64</dt>
                <dd className="long-value">{result.hash_base64}</dd>
              </div>
              <div>
                <dt>Signer type</dt>
                <dd>{result.signer_type}</dd>
              </div>
              <div>
                <dt>Created by user ID</dt>
                <dd>{result.created_by_user_id ?? "Not returned"}</dd>
              </div>
              <div>
                <dt>Created at</dt>
                <dd>{result.created_at}</dd>
              </div>
              <div>
                <dt>Message ID</dt>
                <dd>{result.message_id}</dd>
              </div>
              <div>
                <dt>Algorithm</dt>
                <dd>{result.algorithm}</dd>
              </div>
            </dl>
          </div>
        ) : (
          <div className="empty-panel inline-panel">
            Request a server-signed message to see the verification payload here.
          </div>
        )}
      </section>
    </div>
  );
}
