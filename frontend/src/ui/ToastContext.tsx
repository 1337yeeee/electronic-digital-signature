import {
  createContext,
  ReactNode,
  useCallback,
  useContext,
  useMemo,
  useState
} from "react";
import { useLocale } from "../locales/LocaleContext";
import type { FeedbackTone } from "./feedback";

type Toast = {
  id: number;
  title: string;
  message: string;
  tone: FeedbackTone;
};

type ToastInput = Omit<Toast, "id">;

type ToastContextValue = {
  pushToast: (toast: ToastInput) => void;
};

const ToastContext = createContext<ToastContextValue | undefined>(undefined);

export function ToastProvider({ children }: { children: ReactNode }) {
  const { t } = useLocale();
  const [toasts, setToasts] = useState<Toast[]>([]);

  const removeToast = useCallback((id: number) => {
    setToasts((current) => current.filter((toast) => toast.id !== id));
  }, []);

  const pushToast = useCallback((toast: ToastInput) => {
    const id = Date.now() + Math.floor(Math.random() * 1000);
    setToasts((current) => [...current, { ...toast, id }]);
    window.setTimeout(() => removeToast(id), 4200);
  }, [removeToast]);

  const value = useMemo(() => ({ pushToast }), [pushToast]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toast-viewport" aria-live="polite" aria-atomic="true">
        {toasts.map((toast) => (
          <div key={toast.id} className={`toast-card toast-${toast.tone}`}>
            <div>
              <strong>{toast.title}</strong>
              <p>{toast.message}</p>
            </div>
            <button className="ghost-button" onClick={() => removeToast(toast.id)}>
              {t("common.dismiss")}
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error("useToast must be used within ToastProvider");
  }
  return context;
}
