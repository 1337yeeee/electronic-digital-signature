import { useLocale } from "../locales/LocaleContext";

export function LoadingPage({ label }: { label?: string }) {
  const { t } = useLocale();

  return (
    <main className="status-page status-loading">
      <div className="status-card">
        <p className="eyebrow">{t("common.pleaseWait")}</p>
        <h1>{label ?? t("loading.defaultLabel")}</h1>
        <p>{t("loading.copy")}</p>
      </div>
    </main>
  );
}
