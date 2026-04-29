import { ApiClientError } from "../api/client";

export type FeedbackTone = "info" | "success" | "warning" | "error";

export type FeedbackMessage = {
  title: string;
  message: string;
  tone: FeedbackTone;
};

const feedbackByCode: Record<string, FeedbackMessage> = {
  invalid_signature: {
    title: "Invalid signature",
    message: "The signature does not match the provided or registered message data.",
    tone: "warning"
  },
  invalid_package: {
    title: "Corrupted package",
    message: "The encrypted package is damaged or does not match the expected format.",
    tone: "warning"
  },
  forbidden: {
    title: "Access denied",
    message: "You do not have permission to open or change this resource.",
    tone: "warning"
  },
  token_expired: {
    title: "Session expired",
    message: "Your token has expired. Please sign in again to continue.",
    tone: "warning"
  },
  unauthorized: {
    title: "Authentication required",
    message: "Please sign in to continue working with the application.",
    tone: "warning"
  },
  invalid_token: {
    title: "Session invalid",
    message: "The current token is no longer valid. Please sign in again.",
    tone: "warning"
  },
  invalid_credentials: {
    title: "Login failed",
    message: "Email or password is incorrect.",
    tone: "error"
  },
  document_not_found: {
    title: "Document not found",
    message: "The requested document no longer exists or is not available to you.",
    tone: "warning"
  },
  public_key_required: {
    title: "Public key required",
    message: "Paste a PEM-encoded public key before saving the profile form.",
    tone: "warning"
  },
  invalid_public_key: {
    title: "Invalid public key",
    message: "The public key must be a supported PEM-encoded public key.",
    tone: "warning"
  }
};

export function describeApiError(error: unknown): FeedbackMessage {
  if (error instanceof ApiClientError) {
    if (error.code && feedbackByCode[error.code]) {
      return feedbackByCode[error.code];
    }

    if (error.status === 404) {
      return {
        title: "Not found",
        message: error.message || "The requested page or resource could not be found.",
        tone: "warning"
      };
    }

    if (error.status >= 500) {
      return {
        title: "Server problem",
        message: "The server could not complete the request right now. Please try again.",
        tone: "error"
      };
    }

    if (error.status >= 400) {
      return {
        title: "Request problem",
        message: error.message,
        tone: "warning"
      };
    }
  }

  return {
    title: "Unexpected problem",
    message: error instanceof Error ? error.message : "Something went wrong.",
    tone: "error"
  };
}
