import { Navigate, Route, Routes } from "react-router-dom";
import { AppLayout } from "./layout/AppLayout";
import { ProtectedRoute } from "../components/ProtectedRoute";
import { DashboardPage } from "../pages/DashboardPage";
import { DocumentDetailsPage } from "../pages/DocumentDetailsPage";
import { DocumentFlowPage } from "../pages/DocumentFlowPage";
import { LoginPage } from "../pages/LoginPage";
import { MyDocumentsPage } from "../pages/MyDocumentsPage";
import { NotFoundPage } from "../pages/NotFoundPage";
import { ProfilePage } from "../pages/ProfilePage";
import { RegisterPage } from "../pages/RegisterPage";
import { ServerSignedMessagePage } from "../pages/ServerSignedMessagePage";
import { UserSignatureVerifyPage } from "../pages/UserSignatureVerifyPage";

export function AppRouter() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/app" replace />} />
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />

      <Route element={<ProtectedRoute />}>
        <Route element={<AppLayout />}>
          <Route path="/app" element={<DashboardPage />} />
          <Route path="/app/documents" element={<MyDocumentsPage />} />
          <Route path="/app/documents/:id" element={<DocumentDetailsPage />} />
          <Route path="/app/documents/flow" element={<DocumentFlowPage />} />
          <Route path="/app/profile" element={<ProfilePage />} />
          <Route path="/app/server-signed-message" element={<ServerSignedMessagePage />} />
          <Route path="/app/signatures/verify" element={<UserSignatureVerifyPage />} />
        </Route>
      </Route>

      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  );
}
