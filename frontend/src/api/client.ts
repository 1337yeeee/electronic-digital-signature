const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

export type ApiEnvelopeError = {
  success?: false;
  error?: {
    code?: string;
    message?: string;
  };
};

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
    const payload = text ? JSON.parse(text) : {};

    if (!response.ok) {
      const apiError = payload as ApiEnvelopeError;
      const message =
        apiError.error?.message ?? `Request failed with status ${response.status}`;
      const error = new ApiClientError(message, response.status, apiError.error?.code);

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
