import { Link } from "react-router-dom";
import { useLocale } from "../locales/LocaleContext";

export function NotFoundPage() {
  const { t } = useLocale();

  return (
    <main className="status-page">
      <div className="status-card">
        <p className="eyebrow">404</p>
        <h1>{t("notFound.title")}</h1>
        <p>{t("notFound.copy")}</p>
        <Link className="primary-link" to="/app">
          {t("notFound.back")}
        </Link>
      </div>
    </main>
  );
}
