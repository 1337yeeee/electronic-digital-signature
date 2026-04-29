import { en, type TranslationKey } from "./en";
import { ru } from "./ru";

export type Locale = "en" | "ru";

type TranslationParams = Record<string, string | number | boolean>;

const dictionaries = { en, ru };
const storageKey = "eds-lab-locale";
const browserLocaleToIntl: Record<Locale, string> = {
  en: "en-GB",
  ru: "ru-RU"
};

function detectInitialLocale(): Locale {
  const stored = typeof window !== "undefined"
    ? window.localStorage.getItem(storageKey)
    : null;

  if (stored === "en" || stored === "ru") {
    return stored;
  }

  if (typeof navigator !== "undefined" && navigator.language.toLowerCase().startsWith("ru")) {
    return "ru";
  }

  return "en";
}

let activeLocale: Locale = detectInitialLocale();

export function getLocale(): Locale {
  return activeLocale;
}

export function setLocale(locale: Locale) {
  activeLocale = locale;
  if (typeof window !== "undefined") {
    window.localStorage.setItem(storageKey, locale);
  }
}

export function translate(
  key: TranslationKey,
  params?: TranslationParams,
  locale: Locale = activeLocale
): string {
  const template = dictionaries[locale][key] ?? en[key] ?? key;

  if (!params) {
    return template;
  }

  return Object.entries(params).reduce((message, [name, value]) => {
    return message.replaceAll(`{${name}}`, String(value));
  }, template);
}

export function formatLocalizedDateTime(
  value?: string,
  locale: Locale = activeLocale,
  fallbackKey: TranslationKey = "common.notAvailable"
): string {
  if (!value) {
    return translate(fallbackKey, undefined, locale);
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(browserLocaleToIntl[locale], {
    dateStyle: "medium",
    timeStyle: "short"
  }).format(date);
}

export function translateStatus(status?: string, locale: Locale = activeLocale): string {
  if (!status) {
    return translate("status.created", undefined, locale);
  }

  const key = `status.${status}` as TranslationKey;
  return dictionaries[locale][key] ?? status;
}

export function translateSignerType(type?: string, locale: Locale = activeLocale): string {
  if (!type) {
    return translate("common.notReturned", undefined, locale);
  }

  const key = `signerType.${type}` as TranslationKey;
  return dictionaries[locale][key] ?? type;
}
