import { DatePipe, NgClass } from '@angular/common';
import { Component, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { ApiService } from '../../core/api.service';
import { AuthService } from '../../core/auth.service';
import { LanguageService } from '../../core/language.service';
import { AttachmentReference, ChatMessage, ConversationSummary, PersonaProfile, UserSettings } from '../../core/models';
import { FormattedMessageComponent } from '../../shared/formatted-message.component';
import { OrsaLogoComponent } from '../../shared/orsa-logo.component';
import { TranslatePipe } from '../../shared/translate.pipe';

@Component({
  selector: 'orsa-chat',
  standalone: true,
  imports: [DatePipe, FormsModule, NgClass, RouterLink, FormattedMessageComponent, OrsaLogoComponent, TranslatePipe],
  template: `
    <main class="chat-app" [class.sidebar-collapsed]="sidebarCollapsed()">
      <div class="sidebar-backdrop" (click)="sidebarCollapsed.set(true)"></div>
      <aside class="sidebar" [class.collapsed]="sidebarCollapsed()">
        <div class="sidebar-top">
          <a class="brand compact" routerLink="/" aria-label="ORSA home"><orsa-logo size="sm" /></a>
          <button class="icon-button" type="button" aria-label="Collapse conversations" (click)="sidebarCollapsed.set(true)">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="18" height="18"><path d="M15 18l-6-6 6-6"/></svg>
          </button>
        </div>
        <button class="new-chat" type="button" (click)="newConversation()">{{ 'chat.newConversation' | translate }}</button>
        <label class="search-field">
          <span>{{ 'chat.search' | translate }}</span>
          <input type="search" [(ngModel)]="query" [placeholder]="'chat.searchPlaceholder' | translate">
        </label>
        <nav class="conversation-list" aria-label="Conversation history">
          @for (conversation of filteredConversations(); track conversation.id) {
            <button type="button" class="conversation" [class.active]="conversation.id === activeThreadId()" (click)="selectConversation(conversation.id)">
              <span>{{ conversation.title }}</span>
              <small>{{ conversation.updatedAt | date:'MMM d' }}</small>
            </button>
          }
        </nav>
        <div class="sidebar-footer">
          <a class="sidebar-settings" routerLink="/settings">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="18" height="18" aria-hidden="true">
              <circle cx="12" cy="12" r="3"/>
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
            </svg>
            <span>{{ 'chat.settings' | translate }}</span>
          </a>
        </div>
      </aside>

      <section class="chat-main">
        <header class="chat-topbar">
          <button class="icon-button" type="button" [attr.aria-label]="sidebarCollapsed() ? 'Open conversations' : 'Hide conversations'" (click)="sidebarCollapsed.set(!sidebarCollapsed())">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="18" height="18"><path d="M4 6h16M4 12h16M4 18h16"/></svg>
          </button>
          <div class="chat-topbar__title">
            <orsa-logo size="sm" [showText]="false" />
            <div>
              <strong>ORSA</strong>
              <span>{{ 'chat.brandTag' | translate }}</span>
            </div>
          </div>
          <div class="quota-pill">{{ 'chat.quota' | translate:{ used: settings().attachmentCountToday, limit: settings().attachmentLimit } }}</div>
        </header>

        @if (!auth.isVerified()) {
          <div class="verify-banner" role="status">
            <span>{{ 'chat.verifyBanner' | translate }}</span>
            <button type="button" class="verify-resend" [disabled]="verifyResending()" (click)="resendVerification()">
              {{ (verifyResending() ? 'verify.resending' : 'verify.resend') | translate }}
            </button>
            @if (verifyResent()) {
              <small>{{ 'verify.resent' | translate }}</small>
            }
          </div>
        }

        <div class="message-stream" aria-live="polite">
          @for (message of messages(); track message.createdAt + message.content) {
            <article class="message" [ngClass]="message.role">
              <div class="avatar" [class.avatar--brand]="message.role === 'assistant'">
                @if (message.role === 'assistant') {
                  <orsa-logo size="sm" [showText]="false" />
                } @else {
                  <span>{{ 'chat.you' | translate }}</span>
                }
              </div>
              <orsa-formatted-message [content]="message.content" [role]="message.role" />
            </article>
          }
          @if (isSending()) {
            <article class="message assistant message-loading" aria-live="polite">
              <div class="avatar avatar--brand">
                <orsa-logo size="sm" [showText]="false" />
              </div>
              <div class="loading-bubble" role="status" [attr.aria-label]="'chat.loading' | translate">
                <span class="sr-only">{{ 'chat.loading' | translate }}</span>
                <span class="loading-pulse" aria-hidden="true">
                  <i></i>
                  <i></i>
                  <i></i>
                </span>
              </div>
            </article>
          }
        </div>

        <section class="upload-panel">
          <label class="dropzone" [class.disabled]="!auth.isVerified()">
            <input type="file" multiple accept="image/*,.pdf" [disabled]="!auth.isVerified()" (change)="onFilesSelected($event)">
            <span>{{ 'chat.dropzone' | translate }}</span>
            <small>{{ 'chat.dropzoneSub' | translate }}</small>
          </label>
          @if (selectedFiles().length) {
            <div class="upload-list">
              @for (file of selectedFiles(); track fileKey(file); let index = $index) {
                <span class="upload-file">
                  <span class="upload-file-name">{{ file.name }}</span>
                  <button type="button" class="upload-remove" aria-label="Remove selected file" (click)="removeSelectedFile(index)">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" width="14" height="14" aria-hidden="true">
                      <path d="M18 6 6 18M6 6l12 12"/>
                    </svg>
                  </button>
                </span>
              }
            </div>
          }
          @if (uploadStatus()) {
            <small class="upload-status">{{ uploadStatus() }}</small>
          }
        </section>

        <form class="composer" (ngSubmit)="send()">
          <textarea id="input" name="message" [(ngModel)]="draft" rows="2" [placeholder]="'chat.composerPlaceholder' | translate"></textarea>
          <button class="button button-primary" type="submit" [disabled]="!auth.isVerified() || isSending() || (!draft.trim() && !selectedFiles().length)">{{ 'chat.send' | translate }}</button>
        </form>

        <footer class="chat-ai-footer">{{ 'chat.aiDisclaimer' | translate }}</footer>
      </section>
    </main>
  `
})
export class ChatComponent implements OnInit {
  private readonly defaultStartMessages = new Set([
    'Tell me what is going on.',
    'Tell me what is going on and I will help route the next step safely.'
  ]);
  private readonly api = inject(ApiService);
  readonly auth = inject(AuthService);
  private readonly lang = inject(LanguageService);

  readonly conversations = signal<ConversationSummary[]>([]);
  readonly messages = signal<ChatMessage[]>([]);
  readonly settings = signal<UserSettings>({
    memoryExtractionEnabled: false,
    remindersEnabled: true,
    attachmentCountToday: 0,
    attachmentLimit: 5
  });
  readonly profile = signal<PersonaProfile>({
    displayName: '',
    location: '',
    summary: '',
    consentStatus: 'disabled',
    workflowBoundary: '',
    boundaryPrompt: ''
  });
  readonly activeThreadId = signal(this.api.getCachedActiveThreadId() || 'thread-001');
  readonly selectedFiles = signal<File[]>([]);
  // Start collapsed on small screens so the drawer does not cover the chat on load.
  readonly sidebarCollapsed = signal(this.isNarrowViewport());
  readonly isSending = signal(false);
  readonly uploadStatus = signal('');
  readonly verifyResending = signal(false);
  readonly verifyResent = signal(false);

  query = '';
  draft = '';
  private messageLoadToken = 0;

  ngOnInit(): void {
    this.api.setCachedActiveThreadId(this.activeThreadId());
    this.api.getConversations().subscribe((items) => {
      const conversations = items.length ? items : [this.defaultConversation(this.activeThreadId())];
      this.conversations.set(conversations);
      if (!conversations.some((conversation) => conversation.id === this.activeThreadId())) {
        this.selectConversation(conversations[0].id);
      }
    });
    this.loadMessages(this.activeThreadId());
    this.api.getSettings().subscribe((settings) => this.settings.set(settings));
    this.api.getProfile().subscribe((profile) => this.profile.set(profile));
  }

  private isNarrowViewport(): boolean {
    return typeof window !== 'undefined' && window.innerWidth <= 980;
  }

  resendVerification(): void {
    this.verifyResending.set(true);
    this.verifyResent.set(false);
    this.auth.resendVerification().subscribe({
      next: () => {
        this.verifyResending.set(false);
        this.verifyResent.set(true);
      },
      error: () => {
        this.verifyResending.set(false);
        this.verifyResent.set(true);
      }
    });
  }

  filteredConversations(): ConversationSummary[] {
    const q = this.query.toLowerCase().trim();
    return this.conversations().filter((item) => !q || item.title.toLowerCase().includes(q));
  }

  selectConversation(threadId: string): void {
    this.activeThreadId.set(threadId);
    this.api.setCachedActiveThreadId(threadId);
    this.loadMessages(threadId);
    if (this.isNarrowViewport()) {
      this.sidebarCollapsed.set(true);
    }
  }

  newConversation(): void {
    const id = `thread-${Date.now()}`;
    const next = this.defaultConversation(id);
    const greeting: ChatMessage = { role: 'assistant', content: this.lang.t('chat.greeting'), createdAt: new Date().toISOString() };
    this.conversations.set([next, ...this.conversations()]);
    this.activeThreadId.set(id);
    this.api.setCachedActiveThreadId(id);
    this.api.cacheConversation(next);
    this.setMessagesForThread(id, [greeting]);
  }

  private loadMessages(threadId: string): void {
    const token = ++this.messageLoadToken;
    const cached = this.api.getCachedMessages(threadId);
    if (cached.length) {
      this.messages.set(this.localizeStartMessage(cached));
    }
    this.api.getMessages(threadId).subscribe((items) => {
      if (token !== this.messageLoadToken || threadId !== this.activeThreadId()) {
        return;
      }
      this.setMessagesForThread(threadId, this.localizeStartMessage(items));
    });
  }

  private defaultConversation(threadId: string): ConversationSummary {
    return {
      id: threadId,
      title: this.lang.t('chat.newConversation').replace('+ ', ''),
      updatedAt: new Date().toISOString()
    };
  }

  private localizeStartMessage(items: ChatMessage[]): ChatMessage[] {
    if (items.length !== 1 || items[0].role !== 'assistant') {
      return items;
    }
    if (!this.defaultStartMessages.has(items[0].content.trim())) {
      return items;
    }
    return [{ ...items[0], content: this.lang.t('chat.greeting') }];
  }

  onFilesSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    const incoming = Array.from(input.files || []);
    const existing = this.selectedFiles();
    const available = Math.max(0, this.settings().attachmentLimit - this.settings().attachmentCountToday - existing.length);
    const seen = new Set(existing.map((file) => this.fileKey(file)));
    const filesToAdd = incoming.filter((file) => {
      const key = this.fileKey(file);
      if (seen.has(key)) {
        return false;
      }
      seen.add(key);
      return true;
    }).slice(0, available);

    this.selectedFiles.set([...existing, ...filesToAdd]);
    this.uploadStatus.set(
      incoming.length > filesToAdd.length
        ? this.lang.t('chat.uploadLimit', { n: this.selectedFiles().length })
        : this.lang.t('chat.uploadQueued', { n: this.selectedFiles().length })
    );
    input.value = '';
  }

  removeSelectedFile(index: number): void {
    const next = this.selectedFiles().filter((_, itemIndex) => itemIndex !== index);
    this.selectedFiles.set(next);
    this.uploadStatus.set(next.length ? this.lang.t('chat.uploadQueued', { n: next.length }) : '');
  }

  fileKey(file: File): string {
    return `${file.name}:${file.size}:${file.lastModified}`;
  }

  send(): void {
    const files = this.selectedFiles();
    const typedContent = this.draft.trim();
    if ((!typedContent && !files.length) || this.isSending()) {
      return;
    }
    const content = typedContent || this.lang.t('chat.attachmentReviewPrompt');
    const threadId = this.activeThreadId();
    const userMessage: ChatMessage = { role: 'user', content, createdAt: new Date().toISOString() };
    this.setMessagesForThread(threadId, [...this.messages(), userMessage]);
    this.rememberConversation(threadId, content);
    this.draft = '';
    this.isSending.set(true);
    if (!files.length) {
      this.sendChatMessage(threadId, content, []);
      return;
    }

    this.uploadStatus.set(this.lang.t('chat.uploading'));
    this.api.uploadAttachments(files).subscribe({
      next: (result) => {
        this.settings.update((current) => ({
          ...current,
          attachmentCountToday: result.quota.used,
          attachmentLimit: result.quota.limit
        }));
        this.uploadStatus.set(this.lang.t('chat.uploaded', { n: result.uploaded }));
        this.selectedFiles.set([]);
        this.sendChatMessage(threadId, content, result.attachments);
      },
      error: () => {
        this.isSending.set(false);
        const current = threadId === this.activeThreadId()
          ? this.messages()
          : this.api.getCachedMessages(threadId);
        this.setMessagesForThread(threadId, [...current, {
          role: 'assistant',
          content: this.lang.t('chat.uploadFail'),
          createdAt: new Date().toISOString()
        }]);
      }
    });
  }

  private sendChatMessage(threadId: string, content: string, attachments: AttachmentReference[]): void {
    this.api.sendMessage(threadId, content, attachments, this.api.toProfileContext(this.profile())).subscribe({
      next: (reply) => {
        const current = threadId === this.activeThreadId()
          ? this.messages()
          : this.api.getCachedMessages(threadId);
        this.setMessagesForThread(threadId, [...current, reply]);
        this.rememberConversation(threadId, content);
        this.isSending.set(false);
      },
      error: () => {
        this.isSending.set(false);
        const current = threadId === this.activeThreadId()
          ? this.messages()
          : this.api.getCachedMessages(threadId);
        this.setMessagesForThread(threadId, [...current, {
          role: 'assistant',
          content: this.lang.t('chat.sendError'),
          createdAt: new Date().toISOString()
        }]);
      }
    });
  }

  private setMessagesForThread(threadId: string, messages: ChatMessage[]): void {
    this.api.cacheMessages(threadId, messages);
    if (threadId === this.activeThreadId()) {
      this.messages.set(messages);
    }
  }

  private rememberConversation(threadId: string, content: string): void {
    const summary: ConversationSummary = {
      id: threadId,
      title: this.makeConversationTitle(content),
      updatedAt: new Date().toISOString()
    };
    const next = [summary, ...this.conversations().filter((conversation) => conversation.id !== threadId)];
    this.conversations.set(next);
    this.api.cacheConversation(summary);
  }

  private makeConversationTitle(content: string): string {
    const title = content.replace(/\s+/g, ' ').trim();
    if (!title) {
      return this.lang.t('chat.newConversation').replace('+ ', '');
    }
    return title.length > 40 ? `${title.slice(0, 40)}...` : title;
  }
}
