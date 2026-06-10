import { HttpClient } from '@angular/common/http';
import { Injectable, computed, inject, signal } from '@angular/core';
import { Observable, map, tap } from 'rxjs';

interface Session {
  userId?: string;
  email: string;
  token: string;
  restoredAt: string;
}

interface AuthResponse {
  userId?: string;
  email?: string;
  token?: string;
  displayName?: string;
  provider?: string;
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

  /** Register a new account and persist the returned session identity. */
  register(email: string, password: string, acceptedLegalVersion: string, memoryExtractionEnabled: boolean): Observable<boolean> {
    return this.http
      .post<AuthResponse>(`${this.apiBase}/auth/register`, { email, password, acceptedLegalVersion, memoryExtractionEnabled })
      .pipe(
        tap((res) => this.persistOrThrow(res, email)),
        map(() => true)
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
            this.persist({ email: res.email, userId: res.userId, token: res.token });
          }
        }),
        map((res) => !!(res.email && res.token))
      );
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
    this.persist({ email: res.email ?? fallbackEmail, userId: res.userId, token: res.token });
  }

  private persist(partial: { email: string; userId?: string; token: string }): void {
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
      return parsed && parsed.email && parsed.token ? parsed : null;
    } catch {
      return null;
    }
  }
}
