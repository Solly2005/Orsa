import { Component, input, computed } from '@angular/core';

type MessageRole = 'user' | 'assistant' | 'system';

type InlineToken = {
  kind: 'text' | 'strong';
  text: string;
};

type MessageBlock =
  | { kind: 'heading'; text: string }
  | { kind: 'paragraph'; inlines: InlineToken[] }
  | { kind: 'list'; ordered: boolean; items: InlineToken[][] };

@Component({
  selector: 'orsa-formatted-message',
  standalone: true,
  styles: [':host { display: block; min-width: 0; }'],
  template: `
    <div class="message-content" [class.message-content--formatted]="role() === 'assistant'" dir="auto">
      @if (role() === 'assistant') {
        @for (block of blocks(); track $index) {
          @switch (block.kind) {
            @case ('heading') {
              <h3>{{ block.text }}</h3>
            }
            @case ('paragraph') {
              <p>
                @for (token of block.inlines; track $index) {
                  @if (token.kind === 'strong') {
                    <strong>{{ token.text }}</strong>
                  } @else {
                    {{ token.text }}
                  }
                }
              </p>
            }
            @case ('list') {
              @if (block.ordered) {
                <ol>
                  @for (item of block.items; track $index) {
                    <li>
                      @for (token of item; track $index) {
                        @if (token.kind === 'strong') {
                          <strong>{{ token.text }}</strong>
                        } @else {
                          {{ token.text }}
                        }
                      }
                    </li>
                  }
                </ol>
              } @else {
                <ul>
                  @for (item of block.items; track $index) {
                    <li>
                      @for (token of item; track $index) {
                        @if (token.kind === 'strong') {
                          <strong>{{ token.text }}</strong>
                        } @else {
                          {{ token.text }}
                        }
                      }
                    </li>
                  }
                </ul>
              }
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

  private parseBlocks(value: string): MessageBlock[] {
    const lines = this.normalize(value).split('\n');
    const blocks: MessageBlock[] = [];
    let paragraph: string[] = [];
    let list: { ordered: boolean; items: InlineToken[][] } | null = null;

    const flushParagraph = () => {
      const text = paragraph.join(' ').replace(/\s+/g, ' ').trim();
      if (text) {
        blocks.push({ kind: 'paragraph', inlines: this.parseInline(text) });
      }
      paragraph = [];
    };

    const flushList = () => {
      if (list && list.items.length) {
        blocks.push({ kind: 'list', ordered: list.ordered, items: list.items });
      }
      list = null;
    };

    for (const rawLine of lines) {
      const line = rawLine.trim();
      if (!line) {
        flushParagraph();
        flushList();
        continue;
      }

      const heading = this.headingText(line);
      if (heading) {
        flushParagraph();
        flushList();
        blocks.push({ kind: 'heading', text: heading });
        continue;
      }

      const bullet = line.match(/^[-*]\s+(.+)$/);
      const numbered = line.match(/^\d+[.)]\s+(.+)$/);
      if (bullet || numbered) {
        flushParagraph();
        const ordered = Boolean(numbered);
        if (!list || list.ordered !== ordered) {
          flushList();
          list = { ordered, items: [] };
        }
        list.items.push(this.parseInline((bullet?.[1] ?? numbered?.[1] ?? '').trim()));
        continue;
      }

      flushList();
      paragraph.push(line);
    }

    flushParagraph();
    flushList();

    return blocks.length ? blocks : [{ kind: 'paragraph', inlines: this.parseInline(value) }];
  }

  private normalize(value: string): string {
    return value
      .replace(/\r\n?/g, '\n')
      .replace(/[ \t]+\n/g, '\n')
      .replace(/\n{3,}/g, '\n\n');
  }

  private headingText(line: string): string {
    const markdownHeading = line.match(/^#{1,4}\s+(.+)$/);
    if (markdownHeading) {
      return this.cleanHeading(markdownHeading[1]);
    }

    const boldHeading = line.match(/^\*\*(.+?)\*\*:?\s*$/);
    if (boldHeading) {
      return this.cleanHeading(boldHeading[1]);
    }

    if (line.endsWith(':') && line.length <= 72 && !/[.!?]\s*:$/u.test(line)) {
      return this.cleanHeading(line.slice(0, -1));
    }

    return '';
  }

  private cleanHeading(value: string): string {
    return value.replace(/\*\*/g, '').replace(/:$/, '').trim();
  }

  private parseInline(value: string): InlineToken[] {
    const tokens: InlineToken[] = [];
    const strongPattern = /\*\*(.+?)\*\*/g;
    let lastIndex = 0;
    let match: RegExpExecArray | null;

    while ((match = strongPattern.exec(value)) !== null) {
      if (match.index > lastIndex) {
        tokens.push({ kind: 'text', text: value.slice(lastIndex, match.index) });
      }
      tokens.push({ kind: 'strong', text: match[1].trim() });
      lastIndex = match.index + match[0].length;
    }

    if (lastIndex < value.length) {
      tokens.push({ kind: 'text', text: value.slice(lastIndex) });
    }

    return tokens.length ? tokens : [{ kind: 'text', text: value }];
  }
}
