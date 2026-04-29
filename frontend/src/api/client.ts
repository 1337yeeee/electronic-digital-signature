import { translate } from "../locales";

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

export type ApiEnvelopeError = {
  success?: false;
  error?:
    | {
        code?: string;
        message?: string;
      }
    | string;
};

type ApiEnvelopeErrorObject = {
  code?: string;
  message?: string;
};

function parsePayload(text: string): unknown {
  if (!text) {
    return {};
  }

  try {
    return JSON.parse(text) as unknown;
  } catch {
    return { error: text };
  }
}

function normalizeApiError(
  payload: unknown,
  status: number
): { message: string; code?: string } {
  const envelope = payload as ApiEnvelopeError | undefined;
  const rawError = envelope?.error;

  if (typeof rawError === "string" && rawError.trim()) {
    return {
      message: rawError,
      code: status === 401 ? "unauthorized" : undefined
    };
  }

  const objectError = rawError as ApiEnvelopeErrorObject | undefined;
  if (objectError?.message) {
    return {
      message: objectError.message,
      code: objectError.code
    };
  }

  if (status === 401) {
    return {
      message: translate("api.authenticationRequired"),
      code: "unauthorized"
    };
  }
  if (status === 403) {
    return {
      message: translate("api.forbidden"),
      code: "forbidden"
    };
  }
  if (status === 404) {
    return {
      message: translate("api.notFound"),
      code: "not_found"
    };
  }

  return {
    message: translate("api.requestFailed", { status })
  };
}

function normalizeUnauthorizedCode(error: { message: string; code?: string }) {
  const normalizedMessage = error.message.toLowerCase();
  if (error.code) {
    return error.code;
  }
  if (normalizedMessage.includes("expired")) {
    return "token_expired";
  }
  if (normalizedMessage.includes("token")) {
    return "invalid_token";
  }
  return "unauthorized";
}

export class ApiClientError extends Error {
  readonly status: number;
  readonly code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.name = "ApiClientError";
    this.status = status;
    this.code = code;
  }
}

type ClientConfig = {
  getAccessToken?: () => string | null;
  onUnauthorized?: () => void;
  onForbidden?: () => void;
};

class ApiClient {
  private readonly baseUrl = apiBaseUrl;
  private getAccessToken?: () => string | null;
  private onUnauthorized?: () => void;
  private onForbidden?: () => void;

  configure(config: ClientConfig) {
    this.getAccessToken = config.getAccessToken;
    this.onUnauthorized = config.onUnauthorized;
    this.onForbidden = config.onForbidden;
  }

  async request<T>(path: string, init?: RequestInit): Promise<T> {
    return this.execute<T>(`${this.baseUrl}${path}`, init);
  }

  async requestAbsolute<T>(path: string, init?: RequestInit): Promise<T> {
    return this.execute<T>(path, init);
  }

  private async execute<T>(url: string, init?: RequestInit): Promise<T> {
    const headers = new Headers(init?.headers);
    const token = this.getAccessToken?.();

    if (token && !headers.has("Authorization")) {
      headers.set("Authorization", `Bearer ${token}`);
    }

    if (init?.body && !(init.body instanceof FormData) && !headers.has("Content-Type")) {
      headers.set("Content-Type", "application/json");
    }

    const response = await fetch(url, {
      ...init,
      headers
    });

    const text = await response.text();
    const payload = parsePayload(text);

    if (!response.ok) {
      const normalizedError = normalizeApiError(payload, response.status);
      const error = new ApiClientError(
        normalizedError.message,
        response.status,
        response.status === 401
          ? normalizeUnauthorizedCode(normalizedError)
          : normalizedError.code
      );

      if (response.status === 401) {
        this.onUnauthorized?.();
      }
      if (response.status === 403) {
        this.onForbidden?.();
      }

      throw error;
    }

    return payload as T;
  }
}

export const apiClient = new ApiClient();
export { apiBaseUrl };
