import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from './auth.service';

/**
 * Protects chat (and any member-only route): guests are sent to sign in, and
 * signed-in but unverified email/password users are sent to the email
 * verification step first — making verification a required onboarding state
 * before chat. Google sign-ins are inherently verified and pass straight through.
 */
export const authGuard: CanActivateFn = (_route, state) => {
  const auth = inject(AuthService);
  const router = inject(Router);

  if (!auth.isLoggedIn()) {
    return router.createUrlTree(['/auth'], { queryParams: { redirect: state.url } });
  }
  if (!auth.isVerified()) {
    return router.createUrlTree(['/verify-email']);
  }
  return true;
};

/** Keeps signed-in users out of sign-in/sign-up screens unless they sign out. */
export const signedOutOnlyGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);

  if (!auth.isLoggedIn()) {
    return true;
  }
  return router.createUrlTree(['/chat']);
};
