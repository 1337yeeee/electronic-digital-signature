import { useLocale } from "../locales/LocaleContext";

export function LocaleSwitcher() {
  const { locale, setLocale, t } = useLocale();

  return (
    <div className="locale-switcher" aria-label={t("common.language")}>
      <span className="meta-label">{t("common.language")}</span>
      <div className="copy-actions">
        <button
          type="button"
          className={locale === "en" ? "secondary-button active-pill" : "secondary-button"}
          onClick={() => setLocale("en")}
        >
          EN
        </button>
        <button
          type="button"
          className={locale === "ru" ? "secondary-button active-pill" : "secondary-button"}
          onClick={() => setLocale("ru")}
        >
          RU
        </button>
      </div>
    </div>
  );
}
