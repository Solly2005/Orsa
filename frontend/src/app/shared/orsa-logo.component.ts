import { Component, Input } from '@angular/core';

/**
 * The ORSA bear mark (from the brand kit). Renders the SVG logo, optionally
 * with the wordmark + tagline. Stroke uses currentColor so it inherits the
 * brand colour from the surrounding context.
 */
@Component({
  selector: 'orsa-logo',
  standalone: true,
  template: `
    <span class="logo" [class.logo--sm]="size === 'sm'" [class.logo--mark-only]="!showText">
      <span class="logo__mark" aria-hidden="true">
        <svg viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg" style="width:100%;height:100%">
          <circle cx="15.5" cy="11" r="5.4" stroke="currentColor" stroke-width="2.3"/>
          <circle cx="32.5" cy="11" r="5.4" stroke="currentColor" stroke-width="2.3"/>
          <path d="M24 6.5 C33.2 6.5 39.5 11 39.5 19.2 C39.5 31 32 38.5 24 42.5 C16 38.5 8.5 31 8.5 19.2 C8.5 11 14.8 6.5 24 6.5 Z" stroke="currentColor" stroke-width="2.3" stroke-linejoin="round"/>
          <path d="M24 31.5 C24 31.5 16.8 26.8 16.8 21.6 C16.8 18.9 18.9 17.2 21 17.2 C22.5 17.2 23.6 18.1 24 19 C24.4 18.1 25.5 17.2 27 17.2 C29.1 17.2 31.2 18.9 31.2 21.6 C31.2 26.8 24 31.5 24 31.5 Z" stroke="currentColor" stroke-width="2.3" stroke-linejoin="round"/>
          <path d="M14 23.5 H19.2 L21.4 19.8 L24 25.4" stroke="currentColor" stroke-width="2.3" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </span>
      @if (showText) {
        <span class="logo__text">
          <span class="logo__word">ORSA</span>
          @if (showTag) {
            <span class="logo__tag">Your Intelligent Health Guardian</span>
          }
        </span>
      }
    </span>
  `
})
export class OrsaLogoComponent {
  @Input() size: 'sm' | 'md' = 'md';
  @Input() showText = true;
  @Input() showTag = false;
}
