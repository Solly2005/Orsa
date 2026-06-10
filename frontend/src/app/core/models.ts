export interface ConversationSummary {
  id: string;
  title: string;
  updatedAt: string;
  deleted?: boolean;
}

export interface ChatMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
  createdAt: string;
}

export interface AttachmentReference {
  id: string;
  fileName: string;
  contentType: string;
  storageUri: string;
  caption?: string;
  summary?: string;
  analysisStatus?: 'readable' | 'unreadable' | 'unavailable' | 'failed' | string;
}

export interface AttachmentUploadResult {
  uploaded: number;
  attachments: AttachmentReference[];
  quota: {
    allowed: boolean;
    used: number;
    limit: number;
    resetAt: string;
    message?: string;
  };
}

export interface UserSettings {
  memoryExtractionEnabled: boolean;
  remindersEnabled: boolean;
  attachmentCountToday: number;
  attachmentLimit: number;
}

export interface PersonaProfile {
  displayName: string;
  location: string;
  summary: string;
  consentStatus: 'enabled' | 'disabled';
  workflowBoundary: string;
  boundaryPrompt: string;
  lastPersonaRunAt?: string;
}

export interface ThreadProfileContext {
  consentStatus: 'enabled' | 'disabled';
  personaSummary: string;
  workflowBoundary: string;
  boundaryPrompt: string;
}

export interface NotificationItem {
  id: string;
  label: string;
  status: 'queued' | 'sent' | 'read';
  createdAt: string;
}
