import {
  createContext,
  ReactNode,
  useCallback,
  useContext,
  useMemo,
  useState
} from "react";
import {
  formatLocalizedDateTime,
  getLocale,
  Locale,
  setLocale as persistLocale,
  translate
} from ".";
import type { TranslationKey } from "./en";

type TranslationParams = Record<string, string | number | boolean>;

type LocaleContextValue = {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: TranslationKey, params?: TranslationParams) => string;
  formatDateTime: (value?: string, fallbackKey?: TranslationKey) => string;
};

const LocaleContext = createContext<LocaleContextValue | undefined>(undefined);

export function LocaleProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(() => getLocale());

  const handleSetLocale = useCallback((nextLocale: Locale) => {
    persistLocale(nextLocale);
    setLocaleState(nextLocale);
  }, []);

  const t = useCallback(
    (key: TranslationKey, params?: TranslationParams) => translate(key, params, locale),
    [locale]
  );

  const formatDateTime = useCallback(
    (value?: string, fallbackKey: TranslationKey = "common.notAvailable") =>
      formatLocalizedDateTime(value, locale, fallbackKey),
    [locale]
  );

  const value = useMemo(
    () => ({ locale, setLocale: handleSetLocale, t, formatDateTime }),
    [formatDateTime, handleSetLocale, locale, t]
  );

  return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>;
}

export function useLocale() {
  const context = useContext(LocaleContext);
  if (!context) {
    throw new Error("useLocale must be used within LocaleProvider");
  }
  return context;
}
