import { HttpClient } from '@angular/common/http';
import { Injectable, computed, inject, signal } from '@angular/core';
import { Observable, map, tap } from 'rxjs';

interface Session {
  userId?: string;
  email: string;
  token: string;
  emailVerified: boolean;
  restoredAt: string;
}

interface AuthResponse {
  userId?: string;
  email?: string;
  token?: string;
  emailVerified?: boolean;
  displayName?: string;
  provider?: string;
  // Set when a password was attached to an existing Google account; no session is
  // issued until the emailed link is opened.
  pendingVerification?: boolean;
}

export interface RegisterOutcome {
  /** True once a session was established (brand-new email/password account). */
  loggedIn: boolean;
  /** True when the user must open an email link before password sign-in works. */
  pendingVerification: boolean;
}

// localStorage key prefixes holding session-scoped / health data, cleared on logout.
const PHI_KEY_PREFIXES = [
  'orsa-session',
  'orsa-chat-history',
  'orsa-attachment',
  'orsa-persona',
  'orsa-display-name',
  'orsa-location',
  'orsa-workflow-boundary',
  'orsa-memory-enabled',
  'orsa-last-persona-run',
  'orsa-legal'
];

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly http = inject(HttpClient);
  private readonly apiBase = '/api';
  private readonly storageKey = 'orsa-session';

  private readonly session = signal<Session | null>(this.loadSession());

  readonly isLoggedIn = computed(() => this.session() !== null);
  readonly email = computed(() => this.session()?.email ?? '');
  /** True once the address is confirmed (Google sign-ins are verified inherently). */
  readonly isVerified = computed(() => this.session()?.emailVerified === true);

  /**
   * Authenticate an existing account. Sets the session on success. Backend
   * errors (bad credentials, server down) propagate so the UI can surface them;
   * there is no silent offline fallback, since a session without a verified
   * token cannot call the authenticated API.
   */
  login(email: string, password: string): Observable<boolean> {
    return this.http.post<AuthResponse>(`${this.apiBase}/auth/login`, { email, password }).pipe(
      tap((res) => this.persistOrThrow(res, email)),
      map(() => true)
    );
  }

  /**
   * Register a new account. A brand-new email/password account is signed in
   * immediately (gated until verified). Setting a password on an existing Google
   * account returns pendingVerification with no session — the user must open the
   * emailed link to activate password sign-in.
   */
  register(email: string, password: string, acceptedLegalVersion: string, memoryExtractionEnabled: boolean): Observable<RegisterOutcome> {
    return this.http
      .post<AuthResponse>(`${this.apiBase}/auth/register`, { email, password, acceptedLegalVersion, memoryExtractionEnabled })
      .pipe(
        tap((res) => {
          if (res.token) {
            this.persistOrThrow(res, email);
          }
        }),
        map((res) => ({ loggedIn: !!res.token, pendingVerification: !!res.pendingVerification }))
      );
  }

  /**
   * Initiate Google OAuth sign-in. Saves the intended post-login destination
   * to sessionStorage, then navigates the browser to the backend redirect
   * endpoint which sends the user to Google's consent screen.
   */
  loginWithGoogle(intendedRedirect = '/chat'): void {
    try {
      sessionStorage.setItem('orsa-oauth-redirect', intendedRedirect);
    } catch {
      /* ignore */
    }
    window.location.href = '/api/auth/google';
  }

  /**
   * Exchange the OAuth2 authorization code (received in the callback URL) for
   * a session by posting it to the backend. Sets the session on success.
   * Unlike login(), this does NOT fall back silently — if the exchange fails
   * the caller should surface an error.
   */
  exchangeGoogleCode(code: string, state: string | null): Observable<boolean> {
    return this.http
      .post<AuthResponse>(`${this.apiBase}/auth/google/exchange`, { code, state })
      .pipe(
        tap((res) => {
          if (res.email && res.token) {
            this.persist({ email: res.email, userId: res.userId, token: res.token, emailVerified: res.emailVerified ?? true });
          }
        }),
        map((res) => !!(res.email && res.token))
      );
  }

  /**
   * Confirm an email address from the link token. The backend returns a fresh
   * session token with email_verified=true, so the session is upgraded in place
   * without requiring the user to sign in again.
   */
  verifyEmail(token: string): Observable<boolean> {
    return this.http.post<AuthResponse>(`${this.apiBase}/auth/verify-email`, { token }).pipe(
      tap((res) => {
        if (res.token) {
          // Keep the existing email/userId if the verify response omits them.
          const current = this.session();
          this.persist({
            email: res.email ?? current?.email ?? '',
            userId: res.userId ?? current?.userId,
            token: res.token,
            emailVerified: true
          });
        }
      }),
      map((res) => !!res.token)
    );
  }

  /** Request a fresh verification email for the current (or given) address. */
  resendVerification(email?: string): Observable<boolean> {
    const address = email ?? this.email();
    return this.http
      .post<{ ok: boolean }>(`${this.apiBase}/auth/resend-verification`, { email: address })
      .pipe(map(() => true));
  }

  logout(): void {
    // Clear the session and all locally cached health data so it does not linger
    // on a shared device or leak into the next account on this browser.
    this.clearLocalData();
    this.session.set(null);
  }

  private persistOrThrow(res: AuthResponse, fallbackEmail: string): void {
    if (!res.token) {
      throw new Error('authentication did not return a session token');
    }
    this.persist({ email: res.email ?? fallbackEmail, userId: res.userId, token: res.token, emailVerified: res.emailVerified ?? false });
  }

  private persist(partial: { email: string; userId?: string; token: string; emailVerified: boolean }): void {
    const session: Session = { ...partial, restoredAt: new Date().toISOString() };
    try {
      localStorage.setItem(this.storageKey, JSON.stringify(session));
    } catch {
      /* ignore */
    }
    this.session.set(session);
  }

  private clearLocalData(): void {
    try {
      const keys: string[] = [];
      for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i);
        if (key && PHI_KEY_PREFIXES.some((prefix) => key.startsWith(prefix))) {
          keys.push(key);
        }
      }
      keys.forEach((key) => localStorage.removeItem(key));
    } catch {
      /* ignore */
    }
  }

  private loadSession(): Session | null {
    try {
      const raw = localStorage.getItem(this.storageKey);
      if (!raw) {
        return null;
      }
      const parsed = JSON.parse(raw) as Session;
      // A session is only usable with a token; ignore legacy tokenless sessions.
      if (!parsed || !parsed.email || !parsed.token) {
        return null;
      }
      // Coerce legacy sessions that predate the verification flag.
      return { ...parsed, emailVerified: parsed.emailVerified === true };
    } catch {
      return null;
    }
  }
}
