import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { AuthService } from '../../core/auth.service';
import { LanguageService } from '../../core/language.service';
import { AppNavComponent } from '../../shared/app-nav.component';
import { TranslatePipe } from '../../shared/translate.pipe';

@Component({
  selector: 'orsa-consent',
  standalone: true,
  imports: [AppNavComponent, FormsModule, RouterLink, TranslatePipe],
  template: `
    <orsa-nav />
    <main class="container legal-page">
      <section class="legal-copy">
        <p class="eyebrow">{{ 'consent.eyebrow' | translate }}</p>
        <h1 class="h1">{{ 'consent.title' | translate }}</h1>
        <p class="lead">{{ 'consent.lead' | translate }}</p>
      </section>

      <section class="consent-grid">
        <article>
          <h2 class="h3">{{ 'consent.terms.title' | translate }}</h2>
          <p>{{ 'consent.terms.body' | translate:{ v: legalVersion } }}</p>
          <ul class="legal-points">
            @for (item of termsPoints; track item) {
              <li>{{ item | translate }}</li>
            }
          </ul>
        </article>
        <article>
          <h2 class="h3">{{ 'consent.privacy.title' | translate }}</h2>
          <p>{{ 'consent.privacy.body' | translate }}</p>
          <ul class="legal-points">
            @for (item of privacyPoints; track item) {
              <li>{{ item | translate }}</li>
            }
          </ul>
        </article>
        <article>
          <h2 class="h3">{{ 'consent.create' | translate }}</h2>
          <p class="auth-switch onboarding-switch">
            {{ 'consent.haveAccount' | translate }}
            <a routerLink="/auth" [queryParams]="authSwitchQueryParams">{{ 'auth.signin' | translate }}</a>
          </p>

          <!-- Legal toggles apply to both sign-up paths (email + Google) -->
          <div class="stacked-form">
            <label class="toggle-row">
              <input type="checkbox" name="terms" [(ngModel)]="acceptedTerms">
              <span>{{ 'consent.acceptTerms' | translate }}</span>
            </label>
            <label class="toggle-row">
              <input type="checkbox" name="memory" [(ngModel)]="memoryEnabled">
              <span>{{ 'consent.memory' | translate }}</span>
            </label>

            @if (!acceptedTerms) {
              <p class="consent-hint">{{ 'consent.hint' | translate }}</p>
            }

            <!-- Google sign-up -->
            <button class="btn btn-google" type="button"
                    [disabled]="!acceptedTerms"
                    (click)="signUpWithGoogle()">
              <svg viewBox="0 0 24 24" width="18" height="18" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
                <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
                <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
                <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84z" fill="#FBBC05"/>
                <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
              </svg>
              {{ 'consent.googleSignup' | translate }}
            </button>

            <div class="form-divider">{{ 'consent.orEmail' | translate }}</div>

            <!-- Email sign-up -->
            <form class="stacked-form" (ngSubmit)="save()">
              <label>{{ 'common.email' | translate }}
                <input type="email" name="email" [(ngModel)]="email" autocomplete="email" required>
              </label>
              <label>{{ 'common.password' | translate }}
                <input type="password" name="password" [(ngModel)]="password" autocomplete="new-password" required>
              </label>
              @if (error()) {
                <p class="form-error">{{ error() }}</p>
              }
              <button class="btn btn-primary" type="submit" [disabled]="!acceptedTerms || busy()">
                {{ (busy() ? 'consent.creating' : 'consent.create') | translate }}
              </button>
            </form>

            @if (saved()) {
              <p class="auth-switch">{{ 'consent.savedMsg' | translate }} <a routerLink="/auth">{{ 'consent.signinLink' | translate }}</a></p>
            }
          </div>
        </article>
      </section>
    </main>
  `
})
export class ConsentComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly lang = inject(LanguageService);

  readonly saved = signal(false);
  readonly busy = signal(false);
  readonly error = signal('');
  readonly legalVersion = '2026-06-09';
  readonly termsPoints = [
    'consent.terms.point1',
    'consent.terms.point2',
    'consent.terms.point3',
    'consent.terms.point4',
    'consent.terms.point5',
    'consent.terms.point6'
  ];
  readonly privacyPoints = [
    'consent.privacy.point1',
    'consent.privacy.point2',
    'consent.privacy.point3',
    'consent.privacy.point4',
    'consent.privacy.point5',
    'consent.privacy.point6'
  ];

  email = '';
  password = '';
  acceptedTerms = false;
  memoryEnabled = false;

  get authSwitchQueryParams(): Record<string, string> | null {
    const redirect = this.route.snapshot.queryParamMap.get('redirect');
    return redirect ? { redirect } : null;
  }

  /** Google OAuth sign-up: persist legal acceptance first, then redirect to Google. */
  signUpWithGoogle(): void {
    if (!this.acceptedTerms) return;
    localStorage.setItem('orsa-legal-version', this.legalVersion);
    localStorage.setItem('orsa-legal-accepted-at', new Date().toISOString());
    // Signal to the callback that this was a sign-up (not just a sign-in).
    try {
      sessionStorage.setItem('orsa-oauth-signup', '1');
    } catch { /* ignore */ }
    this.auth.loginWithGoogle(this.postAuthRedirect());
  }

  save(): void {
    if (!this.acceptedTerms) {
      return;
    }
    if (!this.email.trim() || !this.password.trim()) {
      this.error.set(this.lang.t('consent.errFields'));
      return;
    }
    this.error.set('');
    this.busy.set(true);
    localStorage.setItem('orsa-legal-version', this.legalVersion);
    localStorage.setItem('orsa-legal-accepted-at', new Date().toISOString());

    this.auth.register(this.email.trim(), this.password, this.legalVersion, this.memoryEnabled).subscribe({
      next: () => {
        this.busy.set(false);
        this.saved.set(true);
        this.router.navigateByUrl(this.postAuthRedirect());
      },
      error: () => {
        this.busy.set(false);
        this.error.set(this.lang.t('consent.errFail'));
      }
    });
  }

  private postAuthRedirect(): string {
    return this.route.snapshot.queryParamMap.get('redirect') || '/chat';
  }
}
