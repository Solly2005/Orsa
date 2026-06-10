import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, catchError, map, of } from 'rxjs';
import {
  AttachmentReference,
  AttachmentUploadResult,
  ChatMessage,
  ConversationSummary,
  NotificationItem,
  PersonaProfile,
  ThreadProfileContext,
  UserSettings
} from './models';

interface ChatHistoryCache {
  activeThreadId?: string;
  conversations: ConversationSummary[];
  messagesByThread: Record<string, ChatMessage[]>;
}

interface StoredSession {
  userId?: unknown;
  email?: unknown;
  token?: unknown;
}

@Injectable({ providedIn: 'root' })
export class ApiService {
  private readonly http = inject(HttpClient);
  private readonly apiBase = '/api';
  private readonly uploadCountKey = 'orsa-attachment-count';
  private readonly uploadDayKey = 'orsa-attachment-count-day';
  private readonly chatHistoryKeyPrefix = 'orsa-chat-history-v1';
  private readonly sessionKey = 'orsa-session';
  private readonly defaultStartMessages = new Set([
    'Tell me what is going on.',
    'Tell me what is going on and I will help route the next step safely.'
  ]);

  getConversations(): Observable<ConversationSummary[]> {
    return this.http.get<ConversationSummary[]>(`${this.apiBase}/conversations`, this.requestOptions()).pipe(
      map((items) => this.mergeConversationSummaries(items)),
      catchError(() => of(this.readChatHistory().conversations))
    );
  }

  getMessages(threadId: string): Observable<ChatMessage[]> {
    const fallback: ChatMessage[] = [
      {
        role: 'assistant',
        content: 'Tell me what is going on and I will help route the next step safely.',
        createdAt: new Date().toISOString()
      }
    ];

    return this.http.get<ChatMessage[]>(`${this.apiBase}/conversations/${threadId}/messages`, this.requestOptions()).pipe(
      map((items) => this.mergeThreadMessages(threadId, items)),
      catchError(() => {
        const cached = this.getCachedMessages(threadId);
        return of(cached.length ? cached : fallback);
      })
    );
  }

  getCachedActiveThreadId(): string | null {
    return this.readChatHistory().activeThreadId || null;
  }

  setCachedActiveThreadId(threadId: string): void {
    this.writeChatHistory({
      ...this.readChatHistory(),
      activeThreadId: threadId
    });
  }

  getCachedMessages(threadId: string): ChatMessage[] {
    return this.readChatHistory().messagesByThread[threadId] ?? [];
  }

  cacheMessages(threadId: string, messages: ChatMessage[]): void {
    const cache = this.readChatHistory();
    this.writeChatHistory({
      ...cache,
      messagesByThread: {
        ...cache.messagesByThread,
        [threadId]: messages
      }
    });
  }

  cacheConversation(summary: ConversationSummary): void {
    const cache = this.readChatHistory();
    const conversations = this.upsertConversationSummary(cache.conversations, summary);
    this.writeChatHistory({
      ...cache,
      conversations
    });
  }

  sendMessage(
    threadId: string,
    content: string,
    attachments: AttachmentReference[] = [],
    profileContext?: ThreadProfileContext
  ): Observable<ChatMessage> {
    // Errors propagate to the caller so the chat surfaces a real failure instead
    // of a fabricated "message saved" reply that the triage service never saw.
    return this.http.post<ChatMessage>(`${this.apiBase}/chat`, { threadId, content, attachments, profileContext }, this.requestOptions());
  }

  uploadAttachments(files: File[]): Observable<AttachmentUploadResult> {
    const form = new FormData();
    for (const file of files) {
      form.append('files', file, file.name);
    }
    return this.http.post<AttachmentUploadResult>(`${this.apiBase}/attachments`, form, this.requestOptions()).pipe(
      map((result) => this.withPersistedUploadUsage(result))
    );
  }

  getSettings(): Observable<UserSettings> {
    return this.http.get<UserSettings>(`${this.apiBase}/settings`, this.requestOptions()).pipe(
      map((settings) => this.withLocalUploadUsage(settings)),
      catchError(() => of({
        memoryExtractionEnabled: localStorage.getItem('orsa-memory-enabled') === 'true',
        remindersEnabled: true,
        attachmentCountToday: this.localUploadUsage(),
        attachmentLimit: 5
      }))
    );
  }

  updateSettings(settings: Partial<UserSettings>): Observable<UserSettings> {
    if (settings.memoryExtractionEnabled !== undefined) {
      localStorage.setItem('orsa-memory-enabled', String(settings.memoryExtractionEnabled));
    }
    return this.http.patch<UserSettings>(`${this.apiBase}/settings`, settings, this.requestOptions()).pipe(
      catchError(() => this.getSettings())
    );
  }

  getProfile(): Observable<PersonaProfile> {
    const consentStatus = localStorage.getItem('orsa-memory-enabled') === 'true' ? 'enabled' : 'disabled';
    const summary = localStorage.getItem('orsa-persona-summary')
      || 'Persona extraction is stored separately from triage and only runs with explicit consent.';
    const workflowBoundary = localStorage.getItem('orsa-workflow-boundary')
      || 'Stored profile context can personalize response style only when consent is enabled. It must not change clinical urgency, diagnosis, or safety escalation.';
    const fallback: PersonaProfile = {
      displayName: localStorage.getItem('orsa-display-name') || 'Alex Morgan',
      location: localStorage.getItem('orsa-location') || 'Cairo, Egypt',
      summary,
      consentStatus,
      workflowBoundary,
      boundaryPrompt: this.buildBoundaryPrompt(consentStatus === 'enabled', summary, workflowBoundary),
      lastPersonaRunAt: localStorage.getItem('orsa-last-persona-run') || undefined
    };

    return this.http.get<PersonaProfile>(`${this.apiBase}/profile`, this.requestOptions()).pipe(
      catchError(() => of(fallback))
    );
  }

  updateProfile(profile: Partial<PersonaProfile>): Observable<PersonaProfile> {
    if (profile.summary !== undefined) {
      localStorage.setItem('orsa-persona-summary', profile.summary);
    }
    if (profile.workflowBoundary !== undefined) {
      localStorage.setItem('orsa-workflow-boundary', profile.workflowBoundary);
    }
    if (profile.consentStatus !== undefined) {
      localStorage.setItem('orsa-memory-enabled', String(profile.consentStatus === 'enabled'));
    }

    const memoryExtractionEnabled = profile.consentStatus === undefined
      ? undefined
      : profile.consentStatus === 'enabled';

    return this.http.patch<PersonaProfile>(`${this.apiBase}/profile`, {
      memoryExtractionEnabled,
      consentStatus: profile.consentStatus,
      summary: profile.summary,
      workflowBoundary: profile.workflowBoundary
    }, this.requestOptions()).pipe(
      catchError(() => this.getProfile())
    );
  }

  toProfileContext(profile: PersonaProfile): ThreadProfileContext {
    const consentEnabled = profile.consentStatus === 'enabled';
    return {
      consentStatus: profile.consentStatus,
      personaSummary: consentEnabled ? profile.summary : '',
      workflowBoundary: consentEnabled ? profile.workflowBoundary : '',
      boundaryPrompt: consentEnabled
        ? profile.boundaryPrompt || this.buildBoundaryPrompt(true, profile.summary, profile.workflowBoundary)
        : this.buildBoundaryPrompt(false, '', '')
    };
  }

  getNotifications(): Observable<NotificationItem[]> {
    const fallback: NotificationItem[] = [
      { id: 'n-1', label: 'Follow-up queued', status: 'queued', createdAt: new Date().toISOString() },
      { id: 'n-2', label: 'Upload reviewed', status: 'read', createdAt: new Date(Date.now() - 3600000).toISOString() }
    ];

    return this.http.get<NotificationItem[]>(`${this.apiBase}/notifications`, this.requestOptions()).pipe(
      catchError(() => of(fallback))
    );
  }

  private buildBoundaryPrompt(consentEnabled: boolean, personaSummary: string, workflowBoundary: string): string {
    if (!consentEnabled) {
      return 'Personalization consent is disabled. Do not use stored persona summary or workflow boundary in this thread.';
    }
    const parts = [
      'User-approved profile context is available for GPT-OSS in this thread.',
      'Use it only to respect communication preferences and workflow boundaries.'
    ];
    if (personaSummary.trim()) {
      parts.push(`Persona summary: ${personaSummary.trim()}`);
    }
    if (workflowBoundary.trim()) {
      parts.push(`Workflow boundary: ${workflowBoundary.trim()}`);
    }
    parts.push('This context is not clinical evidence. Do not infer symptoms, history, risk, diagnoses, or severity from it. Never let it reduce urgency, override safety rules, or bypass escalation.');
    return parts.join(' ');
  }

  private withPersistedUploadUsage(result: AttachmentUploadResult): AttachmentUploadResult {
    const current = this.localUploadUsage();
    const uploaded = Math.max(0, Number(result.uploaded || 0));
    const backendUsed = Math.max(0, Number(result.quota?.used || 0));
    const used = Math.max(current + uploaded, backendUsed);
    this.saveUploadUsage(used);
    return {
      ...result,
      quota: {
        ...result.quota,
        used
      }
    };
  }

  private withLocalUploadUsage(settings: UserSettings): UserSettings {
    const localUsed = this.localUploadUsage();
    const backendUsed = Math.max(0, Number(settings.attachmentCountToday || 0));
    const attachmentCountToday = Math.max(localUsed, backendUsed);
    if (attachmentCountToday !== localUsed) {
      this.saveUploadUsage(attachmentCountToday);
    }
    return {
      ...settings,
      attachmentCountToday
    };
  }

  private localUploadUsage(): number {
    try {
      if (localStorage.getItem(this.uploadDayKey) !== this.todayKey()) {
        localStorage.setItem(this.uploadDayKey, this.todayKey());
        localStorage.setItem(this.uploadCountKey, '0');
        return 0;
      }
      return Math.max(0, Number(localStorage.getItem(this.uploadCountKey) || '0'));
    } catch {
      return 0;
    }
  }

  private saveUploadUsage(count: number): void {
    try {
      localStorage.setItem(this.uploadDayKey, this.todayKey());
      localStorage.setItem(this.uploadCountKey, String(Math.max(0, count)));
    } catch {
      /* ignore */
    }
  }

  private todayKey(): string {
    return new Date().toISOString().slice(0, 10);
  }

  private mergeConversationSummaries(serverItems: ConversationSummary[]): ConversationSummary[] {
    const cache = this.readChatHistory();
    const conversations = this.mergeConversationLists(serverItems, cache.conversations);
    conversations.sort((a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt));
    this.writeChatHistory({ ...cache, conversations });
    return conversations;
  }

  private mergeThreadMessages(threadId: string, serverMessages: ChatMessage[]): ChatMessage[] {
    const cached = this.getCachedMessages(threadId);
    if (!cached.length) {
      if (!this.isOnlyDefaultStartMessage(serverMessages)) {
        this.cacheMessages(threadId, serverMessages);
      }
      return serverMessages;
    }

    if (!serverMessages.length || this.isOnlyDefaultStartMessage(serverMessages)) {
      return cached;
    }

    const merged = [...serverMessages];
    for (const cachedMessage of cached) {
      if (!this.hasEquivalentMessage(merged, cachedMessage)) {
        merged.push(cachedMessage);
      }
    }
    merged.sort((a, b) => Date.parse(a.createdAt) - Date.parse(b.createdAt));
    this.cacheMessages(threadId, merged);
    return merged;
  }

  private mergeConversationLists(primary: ConversationSummary[], secondary: ConversationSummary[]): ConversationSummary[] {
    const byID = new Map<string, ConversationSummary>();
    for (const item of [...secondary, ...primary]) {
      const existing = byID.get(item.id);
      if (!existing || Date.parse(item.updatedAt) >= Date.parse(existing.updatedAt)) {
        byID.set(item.id, item);
      }
    }
    return [...byID.values()];
  }

  private isOnlyDefaultStartMessage(messages: ChatMessage[]): boolean {
    return messages.length === 1
      && messages[0].role === 'assistant'
      && this.defaultStartMessages.has(messages[0].content.trim());
  }

  private hasEquivalentMessage(messages: ChatMessage[], candidate: ChatMessage): boolean {
    return messages.some((message) => {
      if (message.role !== candidate.role || message.content.trim() !== candidate.content.trim()) {
        return false;
      }
      const messageTime = Date.parse(message.createdAt);
      const candidateTime = Date.parse(candidate.createdAt);
      return Number.isNaN(messageTime)
        || Number.isNaN(candidateTime)
        || Math.abs(messageTime - candidateTime) < 60000;
    });
  }

  private upsertConversationSummary(items: ConversationSummary[], summary: ConversationSummary): ConversationSummary[] {
    const next = [summary, ...items.filter((item) => item.id !== summary.id)];
    next.sort((a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt));
    return next;
  }

  private readChatHistory(): ChatHistoryCache {
    let merged = this.emptyChatHistory();
    for (const key of this.chatHistoryKeys()) {
      try {
        const raw = localStorage.getItem(key);
        if (!raw) {
          continue;
        }
        merged = this.mergeChatHistory(merged, this.parseChatHistory(raw));
      } catch {
        continue;
      }
    }
    return merged;
  }

  private mergeChatHistory(primary: ChatHistoryCache, secondary: ChatHistoryCache): ChatHistoryCache {
    const messagesByThread: Record<string, ChatMessage[]> = { ...primary.messagesByThread };
    for (const [threadId, messages] of Object.entries(secondary.messagesByThread)) {
      const existing = messagesByThread[threadId] ?? [];
      const merged = [...existing];
      for (const message of messages) {
        if (!this.hasEquivalentMessage(merged, message)) {
          merged.push(message);
        }
      }
      messagesByThread[threadId] = merged;
    }
    return {
      activeThreadId: primary.activeThreadId || secondary.activeThreadId,
      conversations: this.mergeConversationLists(primary.conversations, secondary.conversations),
      messagesByThread
    };
  }

  private parseChatHistory(raw: string): ChatHistoryCache {
    try {
      const parsed = JSON.parse(raw) as Partial<ChatHistoryCache>;
      return {
        activeThreadId: typeof parsed.activeThreadId === 'string' ? parsed.activeThreadId : undefined,
        conversations: Array.isArray(parsed.conversations) ? parsed.conversations : [],
        messagesByThread: parsed.messagesByThread && typeof parsed.messagesByThread === 'object'
          ? parsed.messagesByThread
          : {}
      };
    } catch {
      return this.emptyChatHistory();
    }
  }

  private writeChatHistory(cache: ChatHistoryCache): void {
    try {
      localStorage.setItem(this.primaryChatHistoryKey(), JSON.stringify(cache));
    } catch {
      /* ignore */
    }
  }

  private primaryChatHistoryKey(): string {
    return `${this.chatHistoryKeyPrefix}:${this.currentSessionCachePartitions()[0]}`;
  }

  private chatHistoryKeys(): string[] {
    const partitioned = [
      ...this.currentSessionCachePartitions(),
      'guest'
    ].map((partition) => `${this.chatHistoryKeyPrefix}:${partition}`);
    return [...new Set([this.primaryChatHistoryKey(), ...partitioned, this.chatHistoryKeyPrefix])];
  }

  private currentSessionCachePartitions(): string[] {
    const identity = this.currentSessionIdentity();
    const partitions = [identity.email, identity.userId]
      .filter((value): value is string => Boolean(value))
      .map((value) => encodeURIComponent(value.trim().toLowerCase()));
    return partitions.length ? [...new Set(partitions)] : ['guest'];
  }

  private requestOptions(): { headers?: HttpHeaders } {
    const token = this.currentSessionIdentity().token;
    return token ? { headers: new HttpHeaders({ Authorization: `Bearer ${token}` }) } : {};
  }

  private currentSessionIdentity(): { userId: string; email: string; token: string } {
    try {
      const raw = localStorage.getItem(this.sessionKey);
      if (!raw) {
        return { userId: '', email: '', token: '' };
      }
      const parsed = JSON.parse(raw) as StoredSession;
      return {
        userId: typeof parsed.userId === 'string' ? parsed.userId.trim() : '',
        email: typeof parsed.email === 'string' ? parsed.email.trim() : '',
        token: typeof parsed.token === 'string' ? parsed.token.trim() : ''
      };
    } catch {
      return { userId: '', email: '', token: '' };
    }
  }

  private emptyChatHistory(): ChatHistoryCache {
    return {
      conversations: [],
      messagesByThread: {}
    };
  }
}
