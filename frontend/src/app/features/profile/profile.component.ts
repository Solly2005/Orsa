import { DatePipe } from '@angular/common';
import { Component, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { PersonaProfile } from '../../core/models';
import { AppNavComponent } from '../../shared/app-nav.component';
import { ApiService } from '../../core/api.service';
import { TranslatePipe } from '../../shared/translate.pipe';

@Component({
  selector: 'orsa-profile',
  standalone: true,
  imports: [AppNavComponent, DatePipe, FormsModule, TranslatePipe],
  template: `
    <orsa-nav />
    <main class="container profile-page">
      <section class="profile-header">
        <div class="avatar large">{{ profile().displayName.charAt(0) }}</div>
        <div>
          <p class="eyebrow">{{ 'profile.eyebrow' | translate }}</p>
          <h1>{{ profile().displayName }}</h1>
          <p>{{ profile().location }}</p>
        </div>
      </section>

      <section class="profile-grid">
        <article>
          <label class="profile-field">
            <span>{{ 'profile.personaSummary' | translate }}</span>
            <textarea [(ngModel)]="summary" rows="6" [placeholder]="'profile.summaryPlaceholder' | translate"></textarea>
          </label>
        </article>
        <article>
          <h2>{{ 'profile.consentStatus' | translate }}</h2>
          <label class="toggle-row profile-toggle">
            <input type="checkbox" [(ngModel)]="consentEnabled" [attr.aria-label]="'profile.consentStatus' | translate">
            <span>
              <strong>{{ consentEnabled ? ('profile.status.enabled' | translate) : ('profile.status.disabled' | translate) }}</strong>
            </span>
          </label>
          <p>{{ 'profile.lastExtraction' | translate }} {{ profile().lastPersonaRunAt ? (profile().lastPersonaRunAt | date:'medium') : ('profile.notRun' | translate) }}</p>
        </article>
        <article>
          <label class="profile-field">
            <span>{{ 'profile.workflowBoundary' | translate }}</span>
            <textarea [(ngModel)]="workflowBoundary" rows="6" [placeholder]="'profile.boundaryPlaceholder' | translate"></textarea>
          </label>
        </article>
      </section>

      <section class="profile-prompt">
        <h2>{{ 'profile.boundaryPrompt' | translate }}</h2>
        <p>{{ boundaryPromptPreview() }}</p>
      </section>

      <div class="profile-actions">
        <button class="btn btn-primary" type="button" [disabled]="isSaving()" (click)="saveProfile()">{{ 'profile.save' | translate }}</button>
        @if (savedNote()) {
          <span>{{ savedNote() | translate }}</span>
        }
      </div>
    </main>
  `
})
export class ProfileComponent implements OnInit {
  private readonly api = inject(ApiService);
  readonly profile = signal<PersonaProfile>({
    displayName: '',
    location: '',
    summary: '',
    consentStatus: 'disabled',
    workflowBoundary: '',
    boundaryPrompt: ''
  });
  readonly isSaving = signal(false);
  readonly savedNote = signal('');

  summary = '';
  workflowBoundary = '';
  consentEnabled = false;

  ngOnInit(): void {
    this.api.getProfile().subscribe((profile) => this.applyProfile(profile));
  }

  saveProfile(): void {
    if (this.isSaving()) {
      return;
    }
    this.isSaving.set(true);
    this.api.updateProfile({
      summary: this.summary,
      workflowBoundary: this.workflowBoundary,
      consentStatus: this.consentEnabled ? 'enabled' : 'disabled'
    }).subscribe((profile) => {
      this.applyProfile(profile);
      this.savedNote.set('profile.saved');
      this.isSaving.set(false);
    });
  }

  boundaryPromptPreview(): string {
    return this.api.toProfileContext({
      ...this.profile(),
      summary: this.summary,
      workflowBoundary: this.workflowBoundary,
      consentStatus: this.consentEnabled ? 'enabled' : 'disabled',
      boundaryPrompt: ''
    }).boundaryPrompt;
  }

  private applyProfile(profile: PersonaProfile): void {
    this.profile.set(profile);
    this.summary = profile.summary;
    this.workflowBoundary = profile.workflowBoundary;
    this.consentEnabled = profile.consentStatus === 'enabled';
  }
}
