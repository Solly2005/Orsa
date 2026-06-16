import { Component, input, computed } from '@angular/core';
import { NgTemplateOutlet } from '@angular/common';

type MessageRole = 'user' | 'assistant' | 'system';

/** A run of inline content within a paragraph, list item, table cell, etc. */
type InlineToken =
  | { kind: 'text'; text: string }
  | { kind: 'strong'; text: string }
  | { kind: 'em'; text: string }
  | { kind: 'code'; text: string }
  | { kind: 'strike'; text: string }
  | { kind: 'link'; text: string; href: string };

type ListItem = { inlines: InlineToken[]; children: ListBlock | null };
type ListBlock = { kind: 'list'; ordered: boolean; items: ListItem[] };

type MessageBlock =
  | { kind: 'heading'; level: number; text: string }
  | { kind: 'paragraph'; inlines: InlineToken[] }
  | { kind: 'quote'; lines: InlineToken[][] }
  | { kind: 'code'; text: string; lang: string }
  | { kind: 'rule' }
  | { kind: 'table'; head: InlineToken[][]; rows: InlineToken[][][] }
  | ListBlock;

/**
 * Renders assistant replies as rich, ChatGPT/Claude-style Markdown: headings,
 * paragraphs, ordered/unordered lists (one level of nesting), fenced code
 * blocks, inline code, bold/italic/strikethrough, links, blockquotes, tables,
 * and horizontal rules. User messages render verbatim (whitespace preserved).
 *
 * Everything is rendered through Angular's template interpolation/property
 * binding (never innerHTML), so content is escaped and link hrefs are sanitized.
 */
@Component({
  selector: 'orsa-formatted-message',
  standalone: true,
  imports: [NgTemplateOutlet],
  styles: [':host { display: block; min-width: 0; }'],
  template: `
    <!-- Reusable inline renderer; context $implicit is an InlineToken[] -->
    <ng-template #inlines let-tokens>
      @for (token of tokens; track $index) {
        @switch (token.kind) {
          @case ('strong') { <strong>{{ token.text }}</strong> }
          @case ('em') { <em>{{ token.text }}</em> }
          @case ('strike') { <s>{{ token.text }}</s> }
          @case ('code') { <code class="md-code-inline">{{ token.text }}</code> }
          @case ('link') {
            <a [href]="token.href" target="_blank" rel="noopener noreferrer nofollow">{{ token.text }}</a>
          }
          @default { {{ token.text }} }
        }
      }
    </ng-template>

    <!-- Reusable list renderer; context $implicit is a ListBlock -->
    <ng-template #listTpl let-list>
      @if (list.ordered) {
        <ol>
          @for (item of list.items; track $index) {
            <li>
              <ng-container [ngTemplateOutlet]="inlines" [ngTemplateOutletContext]="{ $implicit: item.inlines }" />
              @if (item.children) {
                <ng-container [ngTemplateOutlet]="listTpl" [ngTemplateOutletContext]="{ $implicit: item.children }" />
              }
            </li>
          }
        </ol>
      } @else {
        <ul>
          @for (item of list.items; track $index) {
            <li>
              <ng-container [ngTemplateOutlet]="inlines" [ngTemplateOutletContext]="{ $implicit: item.inlines }" />
              @if (item.children) {
                <ng-container [ngTemplateOutlet]="listTpl" [ngTemplateOutletContext]="{ $implicit: item.children }" />
              }
            </li>
          }
        </ul>
      }
    </ng-template>

    <div class="message-content" [class.message-content--formatted]="role() === 'assistant'" dir="auto">
      @if (role() === 'assistant') {
        @for (block of blocks(); track $index) {
          @switch (block.kind) {
            @case ('heading') {
              @if (block.level <= 2) {
                <h3 class="md-h md-h2">{{ block.text }}</h3>
              } @else {
                <h4 class="md-h md-h4">{{ block.text }}</h4>
              }
            }
            @case ('paragraph') {
              <p><ng-container [ngTemplateOutlet]="inlines" [ngTemplateOutletContext]="{ $implicit: block.inlines }" /></p>
            }
            @case ('list') {
              <ng-container [ngTemplateOutlet]="listTpl" [ngTemplateOutletContext]="{ $implicit: block }" />
            }
            @case ('code') {
              <pre class="md-code-block"><code>{{ block.text }}</code></pre>
            }
            @case ('quote') {
              <blockquote class="md-quote">
                @for (line of block.lines; track $index) {
                  <p><ng-container [ngTemplateOutlet]="inlines" [ngTemplateOutletContext]="{ $implicit: line }" /></p>
                }
              </blockquote>
            }
            @case ('rule') {
              <hr class="md-rule" />
            }
            @case ('table') {
              <div class="md-table-wrap">
                <table class="md-table">
                  <thead>
                    <tr>
                      @for (cell of block.head; track $index) {
                        <th><ng-container [ngTemplateOutlet]="inlines" [ngTemplateOutletContext]="{ $implicit: cell }" /></th>
                      }
                    </tr>
                  </thead>
                  <tbody>
                    @for (row of block.rows; track $index) {
                      <tr>
                        @for (cell of row; track $index) {
                          <td><ng-container [ngTemplateOutlet]="inlines" [ngTemplateOutletContext]="{ $implicit: cell }" /></td>
                        }
                      </tr>
                    }
                  </tbody>
                </table>
              </div>
            }
          }
        }
      } @else {
        <p>{{ content() }}</p>
      }
    </div>
  `
})
export class FormattedMessageComponent {
  readonly content = input.required<string>();
  readonly role = input.required<MessageRole>();

  readonly blocks = computed(() => this.parseBlocks(this.content()));

  // ── Block parsing ──────────────────────────────────────────────────────────

  private parseBlocks(value: string): MessageBlock[] {
    const lines = this.normalize(value).split('\n');
    const blocks: MessageBlock[] = [];
    let i = 0;

    while (i < lines.length) {
      const raw = lines[i];
      const line = raw.trim();

      // Blank line: paragraph/section separator.
      if (!line) { i++; continue; }

      // Fenced code block: ``` or ~~~ optionally followed by a language tag.
      const fence = line.match(/^(```+|~~~+)\s*(\S*)\s*$/);
      if (fence) {
        const marker = fence[1][0];
        const lang = fence[2] ?? '';
        const body: string[] = [];
        i++;
        while (i < lines.length && !new RegExp(`^\\s*${marker === '`' ? '```+' : '~~~+'}\\s*$`).test(lines[i])) {
          body.push(lines[i]);
          i++;
        }
        i++; // skip the closing fence
        blocks.push({ kind: 'code', text: body.join('\n'), lang });
        continue;
      }

      // Horizontal rule: 3+ of - * _ (allowing spaces between).
      if (/^(\s*[-*_]\s*){3,}$/.test(line) && /^[-*_\s]+$/.test(line)) {
        blocks.push({ kind: 'rule' });
        i++;
        continue;
      }

      // Blockquote: one or more consecutive lines starting with >.
      if (line.startsWith('>')) {
        const quoteLines: InlineToken[][] = [];
        while (i < lines.length && lines[i].trim().startsWith('>')) {
          const inner = lines[i].trim().replace(/^>+\s?/, '');
          if (inner) quoteLines.push(this.parseInline(inner));
          i++;
        }
        blocks.push({ kind: 'quote', lines: quoteLines });
        continue;
      }

      // Table: a header row containing "|" followed by a separator row.
      if (line.includes('|') && i + 1 < lines.length && this.isTableSeparator(lines[i + 1])) {
        const head = this.splitTableRow(line).map((c) => this.parseInline(c));
        i += 2;
        const rows: InlineToken[][][] = [];
        while (i < lines.length && lines[i].includes('|') && lines[i].trim()) {
          rows.push(this.splitTableRow(lines[i].trim()).map((c) => this.parseInline(c)));
          i++;
        }
        blocks.push({ kind: 'table', head, rows });
        continue;
      }

      // List (handles one level of indentation-based nesting).
      if (this.matchListItem(raw)) {
        const { block, next } = this.parseList(lines, i);
        blocks.push(block);
        i = next;
        continue;
      }

      // ATX / bold / colon headings.
      const heading = this.headingFor(line);
      if (heading) {
        blocks.push(heading);
        i++;
        continue;
      }

      // Paragraph: gather consecutive "plain" lines.
      const paragraph: string[] = [];
      while (i < lines.length) {
        const cur = lines[i];
        const trimmed = cur.trim();
        if (!trimmed) break;
        if (/^(```+|~~~+)/.test(trimmed)) break;
        if (trimmed.startsWith('>')) break;
        if (this.matchListItem(cur)) break;
        if (this.headingFor(trimmed)) break;
        if (cur.includes('|') && i + 1 < lines.length && this.isTableSeparator(lines[i + 1])) break;
        if (/^(\s*[-*_]\s*){3,}$/.test(trimmed) && /^[-*_\s]+$/.test(trimmed)) break;
        paragraph.push(trimmed);
        i++;
      }
      const text = paragraph.join(' ').replace(/\s+/g, ' ').trim();
      if (text) blocks.push({ kind: 'paragraph', inlines: this.parseInline(text) });
    }

    return blocks.length ? blocks : [{ kind: 'paragraph', inlines: this.parseInline(value) }];
  }

  /** Parses a contiguous run of list items starting at `start`. */
  private parseList(lines: string[], start: number): { block: ListBlock; next: number } {
    const first = this.matchListItem(lines[start])!;
    const baseIndent = first.indent;
    const ordered = first.ordered;
    const items: ListItem[] = [];
    let i = start;

    while (i < lines.length) {
      const m = this.matchListItem(lines[i]);
      if (!m) {
        // A blank line ends the list unless the next non-blank line is a deeper item.
        if (lines[i].trim() === '') {
          const lookahead = this.matchListItem(lines[i + 1] ?? '');
          if (lookahead) { i++; continue; }
        }
        break;
      }

      if (m.indent > baseIndent + 1) {
        // Nested item: attach to the previous top-level item.
        const parent = items[items.length - 1];
        if (parent) {
          if (!parent.children) parent.children = { kind: 'list', ordered: m.ordered, items: [] };
          parent.children.items.push({ inlines: this.parseInline(m.text), children: null });
          i++;
          continue;
        }
      }

      if (m.ordered !== ordered && m.indent <= baseIndent + 1) {
        // A sibling list of a different type starts here.
        break;
      }

      items.push({ inlines: this.parseInline(m.text), children: null });
      i++;
    }

    return { block: { kind: 'list', ordered, items }, next: i };
  }

  private matchListItem(line: string): { indent: number; ordered: boolean; text: string } | null {
    const m = line.match(/^(\s*)([-*+]|\d+[.)])\s+(.+)$/);
    if (!m) return null;
    return {
      indent: m[1].replace(/\t/g, '  ').length,
      ordered: /\d/.test(m[2]),
      text: m[3].trim()
    };
  }

  private headingFor(line: string): { kind: 'heading'; level: number; text: string } | null {
    const atx = line.match(/^(#{1,6})\s+(.+?)\s*#*\s*$/);
    if (atx) return { kind: 'heading', level: atx[1].length, text: this.cleanHeading(atx[2]) };

    const bold = line.match(/^\*\*(.+?)\*\*:?\s*$/);
    if (bold) return { kind: 'heading', level: 3, text: this.cleanHeading(bold[1]) };

    // A short, sentence-free line ending in a colon reads as a section label.
    if (line.endsWith(':') && line.length <= 72 && !/[.!?]\s*:$/u.test(line) && !line.includes('|')) {
      return { kind: 'heading', level: 4, text: this.cleanHeading(line.slice(0, -1)) };
    }
    return null;
  }

  private cleanHeading(value: string): string {
    return value.replace(/\*\*/g, '').replace(/`/g, '').replace(/:$/, '').trim();
  }

  private isTableSeparator(line: string): boolean {
    const t = line.trim();
    if (!t.includes('-') || !t.includes('|')) return false;
    return /^\|?\s*:?-{1,}:?\s*(\|\s*:?-{1,}:?\s*)+\|?$/.test(t);
  }

  private splitTableRow(line: string): string[] {
    let t = line.trim();
    if (t.startsWith('|')) t = t.slice(1);
    if (t.endsWith('|')) t = t.slice(0, -1);
    return t.split(/(?<!\\)\|/).map((c) => c.replace(/\\\|/g, '|').trim());
  }

  private normalize(value: string): string {
    return value
      .replace(/\r\n?/g, '\n')
      .replace(/[ \t]+\n/g, '\n')
      .replace(/\n{3,}/g, '\n\n');
  }

  // ── Inline parsing ─────────────────────────────────────────────────────────

  // Earliest-match alternation: inline code | bold (** or __) | strike (~~)
  // | italic (* or _) | link [text](href). Non-matching spans become text.
  private static readonly INLINE = new RegExp(
    [
      '(`+)([\\s\\S]+?)\\1',                 // 1,2 inline code
      '\\*\\*([^*]+?)\\*\\*',                // 3   **bold**
      '__([^_]+?)__',                        // 4   __bold__
      '~~([^~]+?)~~',                        // 5   ~~strike~~
      '\\*([^*\\n]+?)\\*',                   // 6   *italic*
      '(?<![\\w])_([^_\\n]+?)_(?![\\w])',    // 7   _italic_
      '\\[([^\\]]+?)\\]\\(([^)\\s]+?)\\)'    // 8,9 [text](href)
    ].join('|'),
    'g'
  );

  private parseInline(value: string): InlineToken[] {
    const tokens: InlineToken[] = [];
    const re = new RegExp(FormattedMessageComponent.INLINE.source, 'g');
    let last = 0;
    let m: RegExpExecArray | null;

    while ((m = re.exec(value)) !== null) {
      if (m.index > last) tokens.push({ kind: 'text', text: value.slice(last, m.index) });

      if (m[2] !== undefined) tokens.push({ kind: 'code', text: m[2] });
      else if (m[3] !== undefined) tokens.push({ kind: 'strong', text: m[3].trim() });
      else if (m[4] !== undefined) tokens.push({ kind: 'strong', text: m[4].trim() });
      else if (m[5] !== undefined) tokens.push({ kind: 'strike', text: m[5].trim() });
      else if (m[6] !== undefined) tokens.push({ kind: 'em', text: m[6].trim() });
      else if (m[7] !== undefined) tokens.push({ kind: 'em', text: m[7].trim() });
      else if (m[8] !== undefined && m[9] !== undefined) {
        const href = m[9].trim();
        if (this.isSafeHref(href)) tokens.push({ kind: 'link', text: m[8].trim(), href });
        else tokens.push({ kind: 'text', text: m[0] });
      }

      last = m.index + m[0].length;
    }

    if (last < value.length) tokens.push({ kind: 'text', text: value.slice(last) });
    return tokens.length ? tokens : [{ kind: 'text', text: value }];
  }

  private isSafeHref(href: string): boolean {
    return /^(https?:\/\/|mailto:|tel:|\/)/i.test(href);
  }
}
