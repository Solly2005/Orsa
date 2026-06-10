import { Injectable, signal } from '@angular/core';

export type ThemePreference = 'light' | 'dark' | 'system';

@Injectable({ providedIn: 'root' })
export class ThemeService {
  private readonly storageKey = 'orsa-theme';
  readonly preference = signal<ThemePreference>(this.loadPreference());

  constructor() {
    this.apply(this.preference());
    const media = window.matchMedia('(prefers-color-scheme: dark)');
    media.addEventListener('change', () => {
      if (this.preference() === 'system') {
        this.apply('system');
      }
    });
  }

  set(next: ThemePreference): void {
    localStorage.setItem(this.storageKey, next);
    this.preference.set(next);
    this.apply(next);
  }

  private loadPreference(): ThemePreference {
    const stored = localStorage.getItem(this.storageKey);
    return stored === 'light' || stored === 'dark' || stored === 'system' ? stored : 'system';
  }

  private apply(pref: ThemePreference): void {
    const resolved = pref === 'system'
      ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
      : pref;
    document.documentElement.setAttribute('data-theme', resolved);
    document.documentElement.style.colorScheme = resolved;
  }
}
