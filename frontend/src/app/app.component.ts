import { Component, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { LanguageService } from './core/language.service';
import { ThemeService } from './core/theme.service';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet],
  templateUrl: './app.component.html',
  styleUrl: './app.component.css'
})
export class AppComponent {
  // Inject at the root so the saved theme and language/text-direction are
  // applied on initial load, before any page-specific component mounts.
  private readonly theme = inject(ThemeService);
  private readonly language = inject(LanguageService);
}
