import { Component, OnInit, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthService } from '../../core/auth.service';
import { LanguageService } from '../../core/language.service';
import { AppNavComponent } from '../../shared/app-nav.component';
import { TranslatePipe } from '../../shared/translate.pipe';

@Component({
  selector: 'orsa-google-callback',
  standalone: true,
  imports: [AppNavComponent, RouterLink, TranslatePipe],
  template: `
    <orsa-nav />
    <main class="auth-page container">
      <section class="auth-panel">
        @if (error()) {
          <div>
            <p class="eyebrow">{{ 'gcb.failTitle' | translate }}</p>
            <h1 class="h1">{{ 'gcb.wrong' | translate }}</h1>
            <p class="form-error">{{ error() }}</p>
          </div>
          <a class="btn btn-secondary" routerLink="/auth">{{ 'gcb.back' | translate }}</a>
        } @else {
          <div>
            <p class="eyebrow">{{ 'gcb.signingin' | translate }}</p>
            <h1 class="h1">{{ 'gcb.completing' | translate }}</h1>
            <p class="auth-sub">{{ 'gcb.wait' | translate }}</p>
          </div>
          <div class="google-callback-spinner" aria-label="Loading"></div>
        }
      </section>
    </main>
  `
})
export class GoogleCallbackComponent implements OnInit {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly lang = inject(LanguageService);

  readonly error = signal('');

  ngOnInit(): void {
    const code = this.route.snapshot.queryParamMap.get('code');
    const state = this.route.snapshot.queryParamMap.get('state');
    const oauthError = this.route.snapshot.queryParamMap.get('error');

    if (oauthError) {
      this.error.set(this.lang.t(oauthError === 'access_denied' ? 'gcb.cancelled' : 'gcb.failGeneric'));
      return;
    }

    if (!code) {
      this.error.set(this.lang.t('gcb.noCode'));
      return;
    }

    this.auth.exchangeGoogleCode(code, state).subscribe({
      next: (ok) => {
        if (!ok) {
          this.error.set(this.lang.t('gcb.noSession'));
          return;
        }
        let redirect = '/chat';
        try {
          redirect = sessionStorage.getItem('orsa-oauth-redirect') || '/chat';
          sessionStorage.removeItem('orsa-oauth-redirect');
        } catch {
          /* ignore */
        }
        this.router.navigateByUrl(redirect);
      },
      error: (err) => {
        const msg = err?.error?.error;
        this.error.set(this.lang.t(
          msg === 'Google OAuth is not configured on this server' ? 'gcb.notEnabled' : 'gcb.failGeneric'
        ));
      }
    });
  }
}
