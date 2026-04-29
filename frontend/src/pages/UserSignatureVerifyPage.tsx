import { FormEvent, useEffect, useRef, useState } from "react";
import { ApiClientError, apiClient } from "../api/client";
import { SecurityNotice } from "../components/SecurityNotice";
import { translate, translateSignerType } from "../locales";
import { useLocale } from "../locales/LocaleContext";
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
    return translate("validation.messageRequired");
  }
  return undefined;
}

function validateSignature(value: string): string | undefined {
  const normalized = value.trim();
  if (!normalized) {
    return translate("validation.signatureRequired");
  }

  try {
    atob(normalized);
  } catch {
    return translate("validation.signatureInvalid");
  }

  return undefined;
}

export function UserSignatureVerifyPage() {
  const { t } = useLocale();
  const { pushToast } = useToast();
  const defaultMessageRef = useRef(t("scenario1.defaultMessage"));
  const [message, setMessage] = useState(defaultMessageRef.current);
  const [signatureBase64, setSignatureBase64] = useState("");
  const [messageError, setMessageError] = useState<string | null>(null);
  const [signatureError, setSignatureError] = useState<string | null>(null);
  const [requestError, setRequestError] = useState<string | null>(null);
  const [result, setResult] = useState<VerifyUserSignatureResponse | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    const previousValue = defaultMessageRef.current;
    const nextValue = t("scenario1.defaultMessage");
    if (message === previousValue) {
      setMessage(nextValue);
    }
    defaultMessageRef.current = nextValue;
  }, [message, t]);

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
          title: t("scenario1.toastVerifiedTitle"),
          message: t("scenario1.toastVerifiedMessage"),
          tone: "success"
        });
      } else {
        pushToast({
          title: t("scenario1.toastInvalidTitle"),
          message: response.error || t("scenario1.toastInvalidMessage"),
          tone: "warning"
        });
      }
    } catch (error) {
      if (error instanceof ApiClientError && error.code === "unauthorized") {
        setRequestError(t("scenario1.requestUnauthorized"));
        pushToast({
          title: t("feedback.unauthorized.title"),
          message: t("scenario1.requestUnauthorized"),
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
        <p className="eyebrow">{t("scenario1.eyebrow")}</p>
        <h2>{t("scenario1.title")}</h2>
        <p>{t("scenario1.copy")}</p>
        <SecurityNotice title={t("scenario1.securityTitle")}>
          {t("scenario1.securityCopy")}
        </SecurityNotice>
      </section>

      <section className="scenario-grid">
        <article className="panel">
          <div className="panel-header">
            <div>
              <h3>{t("scenario1.verifyTitle")}</h3>
              <p>{t("scenario1.verifyCopy")}</p>
            </div>
          </div>

          {requestError ? (
            <div className="inline-error" role="alert">
              {requestError}
            </div>
          ) : null}

          <form onSubmit={handleSubmit} noValidate>
            <label>
              {t("scenario1.message")}
              <textarea
                rows={5}
                value={message}
                onChange={(event) => setMessage(event.target.value)}
                autoComplete="off"
                autoCapitalize="off"
                spellCheck={false}
                onBlur={() => setMessageError(validateMessage(message) ?? null)}
                placeholder={t("scenario1.messagePlaceholder")}
              />
              {messageError ? <span className="field-error">{messageError}</span> : null}
            </label>

            <label>
              {t("scenario1.signature")}
              <textarea
                rows={7}
                value={signatureBase64}
                onChange={(event) => setSignatureBase64(event.target.value)}
                autoComplete="off"
                autoCapitalize="off"
                spellCheck={false}
                onBlur={() =>
                  setSignatureError(validateSignature(signatureBase64) ?? null)
                }
                placeholder={t("scenario1.signaturePlaceholder")}
              />
              <span className="field-hint">{t("scenario1.signatureHint")}</span>
              {signatureError ? <span className="field-error">{signatureError}</span> : null}
            </label>

            <div className="form-actions-row">
              <button type="submit" disabled={isSubmitting}>
                {isSubmitting ? t("scenario1.verifyingButton") : t("scenario1.verifyButton")}
              </button>
            </div>
          </form>
        </article>

        <article className="panel">
          <div className="panel-header">
            <div>
              <h3>{t("scenario1.howToTitle")}</h3>
              <p>{t("scenario1.howToCopy")}</p>
            </div>
          </div>

          <ol className="steps-list">
            <li>{t("scenario1.step1")}</li>
            <li>{t("scenario1.step2")}</li>
            <li>{t("scenario1.step3")}</li>
            <li>{t("scenario1.step4")}</li>
          </ol>

          <pre className="code-block">{`printf '%s' '${t("scenario1.defaultMessage").replaceAll("'", "'\\''")}' > message.txt
openssl dgst -sha256 -sign data/user-keys/user_private.pem -out signature.bin message.txt
openssl base64 -A -in signature.bin`}</pre>

          <p className="field-hint">{t("scenario1.matchHint")}</p>
        </article>
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h3>{t("scenario1.resultTitle")}</h3>
            <p>{t("scenario1.resultCopy")}</p>
          </div>
        </div>

        {result ? (
          <div className="result-stack">
            <div className={result.valid ? "result-chip success" : "result-chip danger"}>
              {result.valid ? t("common.valid") : t("common.invalid")}
            </div>
            <dl className="details-list">
              <div>
                <dt>{t("scenario1.resultValid")}</dt>
                <dd>{String(result.valid)}</dd>
              </div>
              <div>
                <dt>{t("scenario1.resultSignerType")}</dt>
                <dd>{translateSignerType(result.signer_type)}</dd>
              </div>
              <div>
                <dt>{t("scenario1.resultSignerUserId")}</dt>
                <dd>{result.signer_user_id ?? t("common.notReturned")}</dd>
              </div>
              <div>
                <dt>{t("scenario1.resultVerifierMessage")}</dt>
                <dd>{result.error ?? t("scenario1.resultSuccess")}</dd>
              </div>
            </dl>
          </div>
        ) : (
          <div className="empty-panel inline-panel">
            {t("scenario1.resultEmpty")}
          </div>
        )}
      </section>
    </div>
  );
}
