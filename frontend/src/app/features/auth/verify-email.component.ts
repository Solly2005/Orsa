import { Component, OnInit, inject, signal } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { AuthService } from '../../core/auth.service';
import { LanguageService } from '../../core/language.service';
import { AppNavComponent } from '../../shared/app-nav.component';
import { TranslatePipe } from '../../shared/translate.pipe';

@Component({
  selector: 'orsa-verify-email',
  standalone: true,
  imports: [AppNavComponent, RouterLink, TranslatePipe],
  template: `
    <orsa-nav />
    <main class="auth-page container">
      <section class="auth-panel">
        @if (status() === 'verifying') {
          <div>
            <p class="eyebrow">{{ 'verify.eyebrow' | translate }}</p>
            <h1 class="h1">{{ 'verify.checking' | translate }}</h1>
            <p class="auth-sub">{{ 'verify.wait' | translate }}</p>
          </div>
          <div class="google-callback-spinner" aria-label="Loading"></div>
        } @else if (status() === 'awaiting') {
          <div>
            <p class="eyebrow">{{ 'verify.eyebrow' | translate }}</p>
            <h1 class="h1">{{ 'verify.awaitingTitle' | translate }}</h1>
            <p class="auth-sub">{{ 'verify.awaitingBody' | translate:{ email: auth.email() } }}</p>
            <p class="auth-sub">{{ 'verify.awaitingHint' | translate }}</p>
          </div>
          <button class="btn btn-secondary" type="button" [disabled]="resending()" (click)="resend()">
            {{ (resending() ? 'verify.resending' : 'verify.resend') | translate }}
          </button>
          @if (resent()) {
            <p class="auth-sub">{{ 'verify.resent' | translate }}</p>
          }
          <button class="btn btn-ghost btn-sm" type="button" (click)="signOut()">{{ 'nav.signOut' | translate }}</button>
        } @else if (status() === 'success') {
          <div>
            <p class="eyebrow">{{ 'verify.eyebrow' | translate }}</p>
            <h1 class="h1">{{ 'verify.successTitle' | translate }}</h1>
            <p class="auth-sub">{{ 'verify.successBody' | translate }}</p>
          </div>
          <a class="btn btn-primary" routerLink="/chat">{{ 'verify.continue' | translate }}</a>
        } @else {
          <div>
            <p class="eyebrow">{{ 'verify.failTitle' | translate }}</p>
            <h1 class="h1">{{ 'verify.failHeading' | translate }}</h1>
            <p class="form-error">{{ error() }}</p>
          </div>
          @if (auth.isLoggedIn()) {
            <button class="btn btn-secondary" type="button" [disabled]="resending()" (click)="resend()">
              {{ (resending() ? 'verify.resending' : 'verify.resend') | translate }}
            </button>
            @if (resent()) {
              <p class="auth-sub">{{ 'verify.resent' | translate }}</p>
            }
          }
          <a class="btn btn-secondary" routerLink="/auth">{{ 'verify.backToSignIn' | translate }}</a>
        }
      </section>
    </main>
  `
})
export class VerifyEmailComponent implements OnInit {
  readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly lang = inject(LanguageService);

  readonly status = signal<'verifying' | 'awaiting' | 'success' | 'error'>('verifying');
  readonly error = signal('');
  readonly resending = signal(false);
  readonly resent = signal(false);

  ngOnInit(): void {
    const token = this.route.snapshot.queryParamMap.get('token');
    if (!token) {
      // No token: this is the onboarding "check your inbox" step reached right
      // after sign-up (or when the guard bounced an unverified user here).
      if (this.auth.isLoggedIn() && this.auth.isVerified()) {
        this.router.navigateByUrl('/chat');
        return;
      }
      if (this.auth.isLoggedIn()) {
        this.status.set('awaiting');
        return;
      }
      this.status.set('error');
      this.error.set(this.lang.t('verify.errMissing'));
      return;
    }

    this.auth.verifyEmail(token).subscribe({
      next: (ok) => {
        if (ok) {
          this.status.set('success');
          // Send verified users straight on after a brief confirmation.
          setTimeout(() => this.router.navigateByUrl('/chat'), 1500);
        } else {
          this.status.set('error');
          this.error.set(this.lang.t('verify.errInvalid'));
        }
      },
      error: () => {
        this.status.set('error');
        this.error.set(this.lang.t('verify.errInvalid'));
      }
    });
  }

  signOut(): void {
    if (!confirm(this.lang.t('nav.signOutConfirm'))) {
      return;
    }
    this.auth.logout();
    this.router.navigate(['/auth']);
  }

  resend(): void {
    this.resending.set(true);
    this.resent.set(false);
    this.auth.resendVerification().subscribe({
      next: () => {
        this.resending.set(false);
        this.resent.set(true);
      },
      error: () => {
        this.resending.set(false);
        this.resent.set(true); // response is intentionally uniform; show the same hint
      }
    });
  }
}
