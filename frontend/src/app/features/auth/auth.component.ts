import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { AuthService } from '../../core/auth.service';
import { LanguageService } from '../../core/language.service';
import { AppNavComponent } from '../../shared/app-nav.component';
import { TranslatePipe } from '../../shared/translate.pipe';

@Component({
  selector: 'orsa-auth',
  standalone: true,
  imports: [AppNavComponent, FormsModule, RouterLink, TranslatePipe],
  template: `
    <orsa-nav />
    <main class="auth-page container">
      <section class="auth-panel">
        <div>
          <p class="eyebrow">{{ 'auth.eyebrow' | translate }}</p>
          <h1 class="h1">{{ 'auth.title' | translate }}</h1>
          <p class="auth-sub">{{ 'auth.sub' | translate }}</p>
        </div>
        <form class="stacked-form" (ngSubmit)="signIn()">
          <label>{{ 'common.email' | translate }}
            <input type="email" name="email" [(ngModel)]="email" autocomplete="email" required>
          </label>
          <label>{{ 'common.password' | translate }}
            <input type="password" name="password" [(ngModel)]="password" autocomplete="current-password" required>
          </label>
          @if (error()) {
            <p class="form-error">{{ error() }}</p>
          }
          <button class="btn btn-primary" type="submit" [disabled]="busy()">
            {{ (busy() ? 'auth.signingin' : 'auth.signin') | translate }}
          </button>
          <div class="form-divider">{{ 'auth.or' | translate }}</div>
          <button class="btn btn-google" type="button" (click)="signInWithGoogle()">
            <svg viewBox="0 0 24 24" width="18" height="18" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
              <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
              <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
              <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84z" fill="#FBBC05"/>
              <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
            </svg>
            {{ 'auth.google' | translate }}
          </button>
          <p class="auth-switch">
            {{ 'auth.newToOrsa' | translate }}
            <a routerLink="/consent" [queryParams]="authSwitchQueryParams">{{ 'auth.createAccountLink' | translate }}</a>
          </p>
        </form>
      </section>

      <section class="auth-panel muted">
        <h2 class="h3">{{ 'auth.whyTitle' | translate }}</h2>
        <ul>
          <li>{{ 'auth.why1' | translate }}</li>
          <li>{{ 'auth.why2' | translate }}</li>
          <li>{{ 'auth.why3' | translate }}</li>
          <li>{{ 'auth.why4' | translate }}</li>
          <li>{{ 'auth.why5' | translate }}</li>
        </ul>
      </section>
    </main>
  `
})
export class AuthComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly lang = inject(LanguageService);

  email = '';
  password = '';
  readonly busy = signal(false);
  readonly error = signal('');

  get authSwitchQueryParams(): Record<string, string> | null {
    const redirect = this.route.snapshot.queryParamMap.get('redirect');
    return redirect ? { redirect } : null;
  }

  signInWithGoogle(): void {
    const redirect = this.route.snapshot.queryParamMap.get('redirect') || '/chat';
    this.auth.loginWithGoogle(redirect);
  }

  signIn(): void {
    if (!this.email.trim() || !this.password.trim()) {
      this.error.set(this.lang.t('auth.errCreds'));
      return;
    }
    this.error.set('');
    this.busy.set(true);
    this.auth.login(this.email.trim(), this.password).subscribe({
      next: () => {
        const redirect = this.route.snapshot.queryParamMap.get('redirect') || '/chat';
        this.router.navigateByUrl(redirect);
      },
      error: () => {
        this.busy.set(false);
        this.error.set(this.lang.t('auth.errFail'));
      }
    });
  }
}
