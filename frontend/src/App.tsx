import { BrowserRouter } from "react-router-dom";
import { AppRouter } from "./app/router";
import { AuthProvider } from "./auth/AuthContext";
import { ToastProvider } from "./ui/ToastContext";

export function App() {
  return (
    <BrowserRouter>
      <ToastProvider>
        <AuthProvider>
          <AppRouter />
        </AuthProvider>
      </ToastProvider>
    </BrowserRouter>
  );
}
