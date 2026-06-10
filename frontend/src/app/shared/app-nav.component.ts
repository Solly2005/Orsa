import { Component, HostListener, inject, signal } from '@angular/core';
import { Router, RouterLink, RouterLinkActive } from '@angular/router';
import { AuthService } from '../core/auth.service';
import { ThemeService } from '../core/theme.service';
import { OrsaLogoComponent } from './orsa-logo.component';
import { TranslatePipe } from './translate.pipe';

@Component({
  selector: 'orsa-nav',
  standalone: true,
  imports: [RouterLink, RouterLinkActive, OrsaLogoComponent, TranslatePipe],
  template: `
    <header class="site-header" [class.scrolled]="scrolled()">
      <div class="container">
        <nav class="nav" aria-label="Primary">
          <div class="nav__links">
            <a routerLink="/chat" routerLinkActive="active">{{ 'nav.chat' | translate }}</a>
            @if (auth.isLoggedIn()) {
              <a routerLink="/profile" routerLinkActive="active">{{ 'nav.profile' | translate }}</a>
              <a routerLink="/settings" routerLinkActive="active">{{ 'nav.settings' | translate }}</a>
            } @else {
              <!-- Consent doubles as the sign-up page; only useful when signed out. -->
              <a routerLink="/consent" routerLinkActive="active">{{ 'nav.consent' | translate }}</a>
            }
          </div>

          <a class="nav__center" routerLink="/" aria-label="ORSA home">
            <orsa-logo size="sm" />
          </a>

          <div class="nav__actions">
            <div class="theme-toggle" role="group" aria-label="Color theme">
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

            @if (auth.isLoggedIn()) {
              <button class="btn btn-ghost btn-sm" type="button" (click)="signOut()">{{ 'nav.signOut' | translate }}</button>
            } @else {
              <a class="nav__login desktop-only" routerLink="/auth">{{ 'nav.login' | translate }}</a>
              <a class="btn btn-primary btn-sm" routerLink="/consent">{{ 'nav.try' | translate }}</a>
            }
          </div>
        </nav>
      </div>
    </header>
  `
})
export class AppNavComponent {
  readonly theme = inject(ThemeService);
  readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  readonly scrolled = signal(false);

  @HostListener('window:scroll')
  onScroll(): void {
    this.scrolled.set(window.scrollY > 8);
  }

  signOut(): void {
    this.auth.logout();
    this.router.navigate(['/']);
  }
}
