import { Pipe, PipeTransform, inject } from '@angular/core';
import { LanguageService } from '../core/language.service';

/**
 * Translates an i18n key in the active language: {{ 'nav.chat' | translate }}.
 * Supports {param} interpolation: {{ 'chat.quota' | translate:{ used: 3, limit: 5 } }}.
 *
 * Marked impure so it re-evaluates on every change-detection pass — this is what
 * lets the whole UI re-render instantly when the user switches language, since
 * the active language is a signal read inside LanguageService.t().
 */
@Pipe({ name: 'translate', standalone: true, pure: false })
export class TranslatePipe implements PipeTransform {
  private readonly lang = inject(LanguageService);

  transform(key: string, params?: Record<string, string | number>): string {
    return this.lang.t(key, params);
  }
}
