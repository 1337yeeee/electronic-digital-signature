import { FormEvent, useState } from "react";
import { ApiClientError, apiClient } from "../api/client";
import { describeApiError } from "../ui/feedback";
import { useToast } from "../ui/ToastContext";

type VerifyUserSignatureResponse = {
  valid: boolean;
  signer_type?: string;
  signer_user_id?: string;
  error?: string;
};

function validateMessage(value: string): string | undefined {
  if (!value.trim()) {
    return "Message is required.";
  }
  return undefined;
}

function validateSignature(value: string): string | undefined {
  const normalized = value.trim();
  if (!normalized) {
    return "Signature base64 is required.";
  }

  try {
    atob(normalized);
  } catch {
    return "Signature must be valid base64.";
  }

  return undefined;
}

export function UserSignatureVerifyPage() {
  const { pushToast } = useToast();
  const [message, setMessage] = useState("I approve the lab scenario and sign this message.");
  const [signatureBase64, setSignatureBase64] = useState("");
  const [messageError, setMessageError] = useState<string | null>(null);
  const [signatureError, setSignatureError] = useState<string | null>(null);
  const [requestError, setRequestError] = useState<string | null>(null);
  const [result, setResult] = useState<VerifyUserSignatureResponse | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();

    const nextMessageError = validateMessage(message) ?? null;
    const nextSignatureError = validateSignature(signatureBase64) ?? null;

    setMessageError(nextMessageError);
    setSignatureError(nextSignatureError);
    setRequestError(null);
    setResult(null);

    if (nextMessageError || nextSignatureError) {
      return;
    }

    setIsSubmitting(true);

    try {
      const response = await apiClient.request<VerifyUserSignatureResponse>(
        "/users/me/signatures/verify",
        {
          method: "POST",
          body: JSON.stringify({
            message: message.trim(),
            signature_base64: signatureBase64.trim()
          })
        }
      );

      setResult(response);
      if (response.valid) {
        pushToast({
          title: "Signature verified",
          message: "The server confirmed the signature for the current user.",
          tone: "success"
        });
      } else {
        pushToast({
          title: "Invalid signature",
          message: response.error || "The signature does not match the provided message.",
          tone: "warning"
        });
      }
    } catch (error) {
      if (error instanceof ApiClientError && error.code === "unauthorized") {
        setRequestError("Please sign in again to verify a user signature.");
        pushToast({
          title: "Authentication required",
          message: "Please sign in again to verify a user signature.",
          tone: "warning"
        });
      } else {
        const feedback = describeApiError(error);
        setRequestError(feedback.message);
        pushToast(feedback);
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="dashboard-grid">
      <section className="content-hero">
        <p className="eyebrow">Scenario 1</p>
        <h2>User signs, server verifies</h2>
        <p>
          Paste the original message and a base64-encoded signature created with
          the private key that matches your registered public key. The backend
          will verify it and return who the signer is.
        </p>
      </section>

      <section className="scenario-grid">
        <article className="panel">
          <div className="panel-header">
            <div>
              <h3>Verify signature</h3>
              <p>Calls `POST /api/v1/users/me/signatures/verify`.</p>
            </div>
          </div>

          {requestError ? (
            <div className="inline-error" role="alert">
              {requestError}
            </div>
          ) : null}

          <form onSubmit={handleSubmit} noValidate>
            <label>
              Message
              <textarea
                rows={5}
                value={message}
                onChange={(event) => setMessage(event.target.value)}
                onBlur={() => setMessageError(validateMessage(message) ?? null)}
                placeholder="Enter the exact message that was signed"
              />
              {messageError ? <span className="field-error">{messageError}</span> : null}
            </label>

            <label>
              Signature base64
              <textarea
                rows={7}
                value={signatureBase64}
                onChange={(event) => setSignatureBase64(event.target.value)}
                onBlur={() =>
                  setSignatureError(validateSignature(signatureBase64) ?? null)
                }
                placeholder="MEUCIQ..."
              />
              <span className="field-hint">
                Paste the signature exactly as base64, without extra JSON or PEM wrappers.
              </span>
              {signatureError ? <span className="field-error">{signatureError}</span> : null}
            </label>

            <div className="form-actions-row">
              <button type="submit" disabled={isSubmitting}>
                {isSubmitting ? "Verifying..." : "Verify signature"}
              </button>
            </div>
          </form>
        </article>

        <article className="panel">
          <div className="panel-header">
            <div>
              <h3>How to get signature base64</h3>
              <p>Minimal flow outside the browser if you have the private key locally.</p>
            </div>
          </div>

          <ol className="steps-list">
            <li>Prepare a text file with the exact message.</li>
            <li>Sign it with the matching private key.</li>
            <li>Convert the binary signature to base64.</li>
            <li>Paste that base64 string into the form on the left.</li>
          </ol>

          <pre className="code-block">{`printf '%s' 'I approve the lab scenario and sign this message.' > message.txt
openssl dgst -sha256 -sign data/user-keys/user_private.pem -out signature.bin message.txt
openssl base64 -A -in signature.bin`}</pre>

          <p className="field-hint">
            The message in the form must match the originally signed bytes exactly.
          </p>
        </article>
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>Verification result</h3>
            <p>This reflects the backend response from the user-signature verify endpoint.</p>
          </div>
        </div>

        {result ? (
          <div className="result-stack">
            <div className={result.valid ? "result-chip success" : "result-chip danger"}>
              {result.valid ? "valid" : "invalid"}
            </div>
            <dl className="details-list">
              <div>
                <dt>Valid</dt>
                <dd>{String(result.valid)}</dd>
              </div>
              <div>
                <dt>Signer type</dt>
                <dd>{result.signer_type ?? "Not returned"}</dd>
              </div>
              <div>
                <dt>Signer user ID</dt>
                <dd>{result.signer_user_id ?? "Not returned"}</dd>
              </div>
              <div>
                <dt>Verifier message</dt>
                <dd>{result.error ?? "Signature verified successfully."}</dd>
              </div>
            </dl>
          </div>
        ) : (
          <div className="empty-panel inline-panel">
            Submit a message and signature to see the verification outcome here.
          </div>
        )}
      </section>
    </div>
  );
}
