import { Component, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { AuthService } from '../../core/auth.service';
import { AppNavComponent } from '../../shared/app-nav.component';
import { OrsaLogoComponent } from '../../shared/orsa-logo.component';
import { TranslatePipe } from '../../shared/translate.pipe';

@Component({
  selector: 'orsa-landing',
  standalone: true,
  imports: [AppNavComponent, RouterLink, OrsaLogoComponent, TranslatePipe],
  template: `
    <orsa-nav />
    <main>
      <section class="hero-band">
        <div class="container hero-grid">
          <div class="hero-copy">
            <p class="eyebrow">{{ 'landing.eyebrow' | translate }}</p>
            <h1 class="h-display">{{ 'landing.meet' | translate }} <span class="accent">ORSA</span></h1>
            <p class="lead">{{ 'landing.lead' | translate }}</p>
            <div class="hero-actions">
              <a class="btn btn-primary btn-lg" routerLink="/chat">{{ (auth.isLoggedIn() ? 'landing.openChat' : 'landing.tryOrsa') | translate }}</a>
              @if (!auth.isLoggedIn()) {
                <a class="btn btn-secondary btn-lg" routerLink="/consent">{{ 'landing.createAccount' | translate }}</a>
              }
            </div>
            <div class="trust-row" aria-label="Product assurances">
              <span class="trust-chip">{{ '✓' }} {{ 'landing.trust.consent' | translate }}</span>
              <span class="trust-chip">{{ '✓' }} {{ 'landing.trust.escalate' | translate }}</span>
              <span class="trust-chip">{{ '✓' }} {{ 'landing.trust.audit' | translate }}</span>
            </div>
          </div>

          <div class="chat-visual" aria-label="ORSA conversation preview">
            <div class="visual-top">
              <span class="visual-brand">
                <orsa-logo size="sm" [showText]="false" />
                {{ 'landing.preview.brand' | translate }}
              </span>
              <span class="status-dot"></span>
            </div>
            <div class="bubble assistant">{{ 'landing.preview.assistant' | translate }}</div>
            <div class="bubble user">{{ 'landing.preview.user' | translate }}</div>
            <div class="alert-strip">{{ 'landing.preview.alert' | translate }}</div>
            <div class="attachment-row">
              <span>{{ 'landing.preview.uploads' | translate }}</span>
              <span>{{ 'landing.preview.ready' | translate }}</span>
            </div>
          </div>
        </div>
      </section>

      <section class="section-band">
        <div class="container feature-grid">
          <article class="feat">
            <span class="feat__ic" aria-hidden="true">{{ '💬' }}</span>
            <h2 class="h3">{{ 'landing.feat1.title' | translate }}</h2>
            <p>{{ 'landing.feat1.desc' | translate }}</p>
          </article>
          <article class="feat">
            <span class="feat__ic" aria-hidden="true">{{ '📄' }}</span>
            <h2 class="h3">{{ 'landing.feat2.title' | translate }}</h2>
            <p>{{ 'landing.feat2.desc' | translate }}</p>
          </article>
          <article class="feat">
            <span class="feat__ic" aria-hidden="true">{{ '🔒' }}</span>
            <h2 class="h3">{{ 'landing.feat3.title' | translate }}</h2>
            <p>{{ 'landing.feat3.desc' | translate }}</p>
          </article>
        </div>
      </section>
    </main>
  `
})
export class LandingComponent {
  readonly auth = inject(AuthService);
}
