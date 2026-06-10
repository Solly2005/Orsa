import { Injectable, signal } from '@angular/core';
import { LangCode, translate } from './i18n';

export interface AppLanguage {
  code: LangCode;
  /** Native-script label shown in the picker. */
  label: string;
  /** Right-to-left scripts (e.g. Arabic) flip the interface direction. */
  rtl?: boolean;
}

/** Languages offered in the settings picker. */
export const APP_LANGUAGES: AppLanguage[] = [
  { code: 'en', label: 'English' },
  { code: 'ar', label: 'العربية', rtl: true },
  { code: 'es', label: 'Español' },
  { code: 'fr', label: 'Français' },
  { code: 'de', label: 'Deutsch' },
  { code: 'hi', label: 'हिन्दी' },
  { code: 'zh', label: '中文' }
];

/**
 * Stores the user's preferred interface language and applies it to the document
 * (sets <html lang> and text direction). Right-to-left languages such as Arabic
 * switch the whole layout to RTL. Persisted in localStorage so it survives reloads.
 */
@Injectable({ providedIn: 'root' })
export class LanguageService {
  private readonly storageKey = 'orsa-language';
  readonly current = signal<LangCode>(this.load());

  constructor() {
    this.apply(this.current());
  }

  set(code: LangCode): void {
    try {
      localStorage.setItem(this.storageKey, code);
    } catch {
      /* ignore */
    }
    this.current.set(code);
    this.apply(code);
  }

  /** Translate a key in the current language, with optional {param} interpolation. */
  t(key: string, params?: Record<string, string | number>): string {
    return translate(this.current(), key, params);
  }

  private load(): LangCode {
    try {
      const stored = localStorage.getItem(this.storageKey);
      if (stored && APP_LANGUAGES.some((l) => l.code === stored)) {
        return stored as LangCode;
      }
    } catch {
      /* ignore */
    }
    return 'en';
  }

  private apply(code: LangCode): void {
    const lang = APP_LANGUAGES.find((l) => l.code === code) ?? APP_LANGUAGES[0];
    document.documentElement.lang = lang.code;
    document.documentElement.dir = lang.rtl ? 'rtl' : 'ltr';
  }
}
