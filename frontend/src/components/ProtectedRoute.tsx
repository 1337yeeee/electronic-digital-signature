import { Navigate, Outlet, useLocation } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { useLocale } from "../locales/LocaleContext";
import { LoadingPage } from "../pages/LoadingPage";

export function ProtectedRoute() {
  const { accessToken, isBootstrapping } = useAuth();
  const { t } = useLocale();
  const location = useLocation();

  if (isBootstrapping) {
    return <LoadingPage label={t("loading.restoreSession")} />;
  }

  if (!accessToken) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }

  return <Outlet />;
}
