import { Component, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { ApiService } from '../../core/api.service';
import { APP_LANGUAGES, LanguageService } from '../../core/language.service';
import { ThemeService } from '../../core/theme.service';
import { UserSettings } from '../../core/models';
import { AppNavComponent } from '../../shared/app-nav.component';
import { TranslatePipe } from '../../shared/translate.pipe';

@Component({
  selector: 'orsa-settings',
  standalone: true,
  imports: [AppNavComponent, FormsModule, RouterLink, TranslatePipe],
  template: `
    <orsa-nav />
    <main class="container settings-page">
      <div class="settings-header">
        <div>
          <p class="eyebrow">{{ 'settings.eyebrow' | translate }}</p>
          <h1 style="margin: 0;">{{ 'settings.title' | translate }}</h1>
        </div>
        <a class="btn btn-secondary btn-sm" routerLink="/chat">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" width="16" height="16" aria-hidden="true" style="margin-inline-end: 6px; display: inline-block; vertical-align: middle;"><path d="M19 12H5"/><path d="M12 19l-7-7 7-7"/></svg>
          <span style="vertical-align: middle;">{{ 'settings.backToChat' | translate }}</span>
        </a>
      </div>

      <section class="settings-list">
        <!-- Color theme -->
        <div class="setting-row">
          <span>
            <strong>{{ 'settings.theme' | translate }}</strong>
            <small>{{ 'settings.themeDesc' | translate }}</small>
          </span>
          <div class="theme-toggle" role="group" [attr.aria-label]="'settings.theme' | translate">
            <button type="button" [attr.aria-pressed]="theme.preference() === 'light'" [attr.aria-label]="'nav.theme.light' | translate" [attr.title]="'nav.theme.light' | translate" (click)="theme.set('light')">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/></svg>
            </button>
            <button type="button" [attr.aria-pressed]="theme.preference() === 'system'" [attr.aria-label]="'nav.theme.system' | translate" [attr.title]="'nav.theme.system' | translate" (click)="theme.set('system')">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="12" rx="2"/><path d="M8 20h8M12 16v4"/></svg>
            </button>
            <button type="button" [attr.aria-pressed]="theme.preference() === 'dark'" [attr.aria-label]="'nav.theme.dark' | translate" [attr.title]="'nav.theme.dark' | translate" (click)="theme.set('dark')">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/></svg>
            </button>
          </div>
        </div>

        <!-- App language -->
        <label class="setting-row">
          <span>
            <strong>{{ 'settings.language' | translate }}</strong>
            <small>{{ 'settings.languageDesc' | translate }}</small>
          </span>
          <select class="setting-select" [ngModel]="language.current()" (ngModelChange)="language.set($event)" [attr.aria-label]="'settings.language' | translate">
            @for (lang of languages; track lang.code) {
              <option [value]="lang.code">{{ lang.label }}</option>
            }
          </select>
        </label>

        <!-- Consent for data gathering -->
        <label class="setting-row">
          <span>
            <strong>{{ 'settings.memory' | translate }}</strong>
            <small>{{ 'settings.memoryDesc' | translate }}</small>
          </span>
          <input type="checkbox" [(ngModel)]="memoryEnabled" (change)="saveMemory()" [attr.aria-label]="'settings.memory' | translate">
        </label>
      </section>

      @if (savedNote()) {
        <p class="settings-saved">{{ savedNote() }}</p>
      }
    </main>
  `
})
export class SettingsComponent implements OnInit {
  private readonly api = inject(ApiService);
  readonly theme = inject(ThemeService);
  readonly language = inject(LanguageService);

  readonly languages = APP_LANGUAGES;
  readonly settings = signal<UserSettings>({
    memoryExtractionEnabled: false,
    remindersEnabled: true,
    attachmentCountToday: 0,
    attachmentLimit: 5
  });
  readonly savedNote = signal('');

  memoryEnabled = false;

  ngOnInit(): void {
    this.api.getSettings().subscribe((settings) => {
      this.settings.set(settings);
      this.memoryEnabled = settings.memoryExtractionEnabled;
    });
  }

  saveMemory(): void {
    this.api.updateSettings({ memoryExtractionEnabled: this.memoryEnabled }).subscribe((settings) => {
      this.settings.set(settings);
      this.savedNote.set(this.language.t(this.memoryEnabled ? 'settings.savedOn' : 'settings.savedOff'));
    });
  }
}
