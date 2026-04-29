import {
  createContext,
  ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState
} from "react";
import { apiClient, ApiClientError } from "../api/client";
import { useLocale } from "../locales/LocaleContext";
import {
  clearStoredSession,
  loadStoredToken,
  loadStoredUser,
  persistSession
} from "./auth-storage";
import type { LoginResponse, User } from "../types/auth";
import { describeApiError } from "../ui/feedback";
import { useToast } from "../ui/ToastContext";

type AuthContextValue = {
  accessToken: string | null;
  currentUser: User | null;
  isBootstrapping: boolean;
  authNotice: string | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  refreshCurrentUser: () => Promise<void>;
  clearAuthNotice: () => void;
};

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const { t } = useLocale();
  const { pushToast } = useToast();
  const [accessToken, setAccessToken] = useState<string | null>(() => loadStoredToken());
  const [currentUser, setCurrentUser] = useState<User | null>(() => loadStoredUser());
  const [isBootstrapping, setIsBootstrapping] = useState(true);
  const [authNotice, setAuthNotice] = useState<string | null>(null);

  const logout = useCallback(() => {
    setAccessToken(null);
    setCurrentUser(null);
    clearStoredSession();
  }, []);

  useEffect(() => {
    apiClient.configure({
      getAccessToken: () => accessToken,
      onUnauthorized: () => {
        logout();
        setAuthNotice(t("auth.noticeSessionExpired"));
        pushToast({
          title: t("auth.toastSessionExpiredTitle"),
          message: t("auth.toastSessionExpiredMessage"),
          tone: "warning"
        });
      },
      onForbidden: () => {
        setAuthNotice(t("auth.noticeForbidden"));
        pushToast({
          title: t("auth.toastForbiddenTitle"),
          message: t("auth.toastForbiddenMessage"),
          tone: "warning"
        });
      }
    });
  }, [accessToken, logout, pushToast, t]);

  const refreshCurrentUser = useCallback(async () => {
    const response = await apiClient.request<{ data: User }>("/auth/me");
    setCurrentUser(response.data);
    if (accessToken) {
      persistSession(accessToken, response.data);
    }
  }, [accessToken]);

  useEffect(() => {
    let cancelled = false;

    async function bootstrapSession() {
      if (!accessToken) {
        setIsBootstrapping(false);
        return;
      }

      try {
        const response = await apiClient.request<{ data: User }>("/auth/me");
        if (cancelled) {
          return;
        }
        setCurrentUser(response.data);
        persistSession(accessToken, response.data);
      } catch (error) {
        if (cancelled) {
          return;
        }
        if (!(error instanceof ApiClientError) || error.status !== 401) {
          const feedback = describeApiError(error);
          setAuthNotice(feedback.message);
          pushToast(feedback);
        }
        logout();
      } finally {
        if (!cancelled) {
          setIsBootstrapping(false);
        }
      }
    }

    void bootstrapSession();

    return () => {
      cancelled = true;
    };
  }, [accessToken, logout, pushToast]);

  const login = useCallback(async (email: string, password: string) => {
    const response = await apiClient.request<LoginResponse>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password })
    });

    setAccessToken(response.data.access_token);
    setCurrentUser(response.data.user);
    setAuthNotice(null);
    persistSession(response.data.access_token, response.data.user);
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      accessToken,
      currentUser,
      isBootstrapping,
      authNotice,
      login,
      logout,
      refreshCurrentUser,
      clearAuthNotice: () => setAuthNotice(null)
    }),
    [accessToken, authNotice, currentUser, isBootstrapping, login, logout, refreshCurrentUser]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}
