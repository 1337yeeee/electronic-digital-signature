import { BrowserRouter } from "react-router-dom";
import { AppRouter } from "./app/router";
import { AuthProvider } from "./auth/AuthContext";
import { LocaleProvider } from "./locales/LocaleContext";
import { ToastProvider } from "./ui/ToastContext";

export function App() {
  return (
    <BrowserRouter>
      <LocaleProvider>
        <ToastProvider>
          <AuthProvider>
            <AppRouter />
          </AuthProvider>
        </ToastProvider>
      </LocaleProvider>
    </BrowserRouter>
  );
}
