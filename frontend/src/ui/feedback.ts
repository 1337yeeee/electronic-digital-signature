import { ApiClientError } from "../api/client";
import { translate } from "../locales";

export type FeedbackTone = "info" | "success" | "warning" | "error";

export type FeedbackMessage = {
  title: string;
  message: string;
  tone: FeedbackTone;
};

function feedbackByCode(): Record<string, FeedbackMessage> {
  return {
    invalid_signature: {
      title: translate("feedback.invalidSignature.title"),
      message: translate("feedback.invalidSignature.message"),
      tone: "warning"
    },
    invalid_package: {
      title: translate("feedback.invalidPackage.title"),
      message: translate("feedback.invalidPackage.message"),
      tone: "warning"
    },
    forbidden: {
      title: translate("feedback.forbidden.title"),
      message: translate("feedback.forbidden.message"),
      tone: "warning"
    },
    token_expired: {
      title: translate("feedback.tokenExpired.title"),
      message: translate("feedback.tokenExpired.message"),
      tone: "warning"
    },
    unauthorized: {
      title: translate("feedback.unauthorized.title"),
      message: translate("feedback.unauthorized.message"),
      tone: "warning"
    },
    invalid_token: {
      title: translate("feedback.invalidToken.title"),
      message: translate("feedback.invalidToken.message"),
      tone: "warning"
    },
    invalid_credentials: {
      title: translate("feedback.invalidCredentials.title"),
      message: translate("feedback.invalidCredentials.message"),
      tone: "error"
    },
    document_not_found: {
      title: translate("feedback.documentNotFound.title"),
      message: translate("feedback.documentNotFound.message"),
      tone: "warning"
    },
    public_key_required: {
      title: translate("feedback.publicKeyRequired.title"),
      message: translate("feedback.publicKeyRequired.message"),
      tone: "warning"
    },
    invalid_public_key: {
      title: translate("feedback.invalidPublicKey.title"),
      message: translate("feedback.invalidPublicKey.message"),
      tone: "warning"
    }
  };
}

export function describeApiError(error: unknown): FeedbackMessage {
  if (error instanceof ApiClientError) {
    const dictionary = feedbackByCode();
    if (error.code && dictionary[error.code]) {
      return dictionary[error.code];
    }

    if (error.status === 404) {
      return {
        title: translate("feedback.notFound.title"),
        message: error.message || translate("feedback.notFound.message"),
        tone: "warning"
      };
    }

    if (error.status >= 500) {
      return {
        title: translate("feedback.serverProblem.title"),
        message: translate("feedback.serverProblem.message"),
        tone: "error"
      };
    }

    if (error.status >= 400) {
      return {
        title: translate("feedback.requestProblem.title"),
        message: error.message,
        tone: "warning"
      };
    }
  }

  return {
    title: translate("feedback.unexpectedProblem.title"),
    message: error instanceof Error ? error.message : translate("feedback.unexpectedProblem.message"),
    tone: "error"
  };
}
