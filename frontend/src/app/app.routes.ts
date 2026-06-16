import { Routes } from '@angular/router';
import { authGuard, signedOutOnlyGuard } from './core/auth.guard';
import { AuthComponent } from './features/auth/auth.component';
import { GoogleCallbackComponent } from './features/auth/google-callback.component';
import { VerifyEmailComponent } from './features/auth/verify-email.component';
import { ChatComponent } from './features/chat/chat.component';
import { ConsentComponent } from './features/consent/consent.component';
import { LandingComponent } from './features/landing/landing.component';
import { ProfileComponent } from './features/profile/profile.component';
import { SettingsComponent } from './features/settings/settings.component';

export const routes: Routes = [
  { path: '', component: LandingComponent, title: 'ORSA - Your Health Companion' },
  { path: 'chat', component: ChatComponent, title: 'ORSA - Chat', canActivate: [authGuard] },
  { path: 'auth', component: AuthComponent, title: 'ORSA - Sign in', canActivate: [signedOutOnlyGuard] },
  { path: 'auth/google/callback', component: GoogleCallbackComponent, title: 'ORSA - Signing in with Google' },
  { path: 'verify-email', component: VerifyEmailComponent, title: 'ORSA - Verify email' },
  { path: 'consent', component: ConsentComponent, title: 'ORSA - Create account', canActivate: [signedOutOnlyGuard] },
  { path: 'profile', component: ProfileComponent, title: 'ORSA - Profile', canActivate: [authGuard] },
  { path: 'settings', component: SettingsComponent, title: 'ORSA - Settings', canActivate: [authGuard] },
  { path: '**', redirectTo: '' }
];
