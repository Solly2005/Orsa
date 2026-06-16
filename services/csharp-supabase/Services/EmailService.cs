using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;

namespace Orsa.SupabaseEngine.Services;

/// <summary>
/// Sends transactional email via the Resend HTTP API. When RESEND_API_KEY is
/// unset the service is a no-op (logs a warning) so local development works
/// without email delivery — the user simply lands unverified.
/// </summary>
public sealed class EmailService(IHttpClientFactory httpFactory, IConfiguration config)
{
    private const string ResendEndpoint = "https://api.resend.com/emails";

    public bool IsConfigured => !string.IsNullOrWhiteSpace(config["RESEND_API_KEY"]);

    /// <summary>
    /// Sends the verification email. Returns true if the provider accepted it.
    /// Never throws: delivery failures are logged and reported as false so
    /// registration still succeeds (login is allowed; features stay gated).
    /// </summary>
    public async Task<bool> SendVerificationEmailAsync(string toEmail, string rawToken, CancellationToken ct)
    {
        var apiKey = config["RESEND_API_KEY"]?.Trim();
        if (string.IsNullOrWhiteSpace(apiKey))
        {
            Console.WriteLine("[warn] RESEND_API_KEY not set; skipping verification email. User remains unverified.");
            return false;
        }

        var from = config["RESEND_FROM"]?.Trim();
        if (string.IsNullOrWhiteSpace(from))
        {
            from = "ORSA <onboarding@resend.dev>";
        }

        var appBase = (config["APP_BASE_URL"]?.Trim() ?? "http://localhost:4200").TrimEnd('/');
        var link = $"{appBase}/verify-email?token={Uri.EscapeDataString(rawToken)}";

        var html =
            "<div style=\"font-family:Segoe UI,Arial,sans-serif;font-size:15px;color:#1a1a1a;line-height:1.6\">" +
            "<p>Welcome to ORSA.</p>" +
            "<p>Please confirm your email address to unlock chat and uploads:</p>" +
            $"<p><a href=\"{link}\" style=\"display:inline-block;padding:10px 18px;background:#0b6e4f;color:#fff;border-radius:8px;text-decoration:none\">Verify my email</a></p>" +
            $"<p>If the button does not work, paste this link into your browser:<br><a href=\"{link}\">{link}</a></p>" +
            "<p style=\"color:#666;font-size:13px\">This link expires in 24 hours. If you did not create an ORSA account, you can safely ignore this email.</p>" +
            "</div>";

        // A plain-text alternative materially improves deliverability and keeps the
        // link usable in text-only clients and stricter spam filters.
        var text =
            "Welcome to ORSA.\n\n" +
            "Please confirm your email address to unlock chat and uploads:\n" +
            link + "\n\n" +
            "This link expires in 24 hours. If you did not create an ORSA account, you can ignore this email.";

        return await SendAsync(apiKey, from, toEmail, "Verify your ORSA email", html, text, link, ct);
    }

    /// <summary>
    /// POSTs a single transactional email to Resend with both HTML and text parts.
    /// Never throws: delivery failures are logged (with the fallback link) and
    /// reported as false so the calling flow can continue.
    /// </summary>
    private async Task<bool> SendAsync(
        string apiKey, string from, string toEmail, string subject,
        string html, string text, string fallbackLink, CancellationToken ct)
    {
        var payload = JsonSerializer.Serialize(new
        {
            from,
            to = new[] { toEmail },
            subject,
            html,
            text
        });

        try
        {
            using var http = httpFactory.CreateClient();
            using var req = new HttpRequestMessage(HttpMethod.Post, ResendEndpoint)
            {
                Content = new StringContent(payload, Encoding.UTF8, "application/json")
            };
            req.Headers.Authorization = new AuthenticationHeaderValue("Bearer", apiKey);

            using var resp = await http.SendAsync(req, ct);
            var body = await resp.Content.ReadAsStringAsync(ct);
            if (resp.IsSuccessStatusCode)
            {
                Console.WriteLine($"[info] Resend accepted email to {Mask(toEmail)} (id: {ExtractId(body)}).");
                return true;
            }

            // 403 with "domain is not verified" / sandbox sender is the usual cause of
            // "emails are not arriving": surface it explicitly so it is actionable.
            Console.WriteLine($"[warn] Resend rejected email to {Mask(toEmail)} ({(int)resp.StatusCode}): {Truncate(body, 400)}");
            Console.WriteLine($"[warn] Check RESEND_FROM uses a verified domain and RESEND_API_KEY is a live key.");
            if (!string.IsNullOrEmpty(fallbackLink))
            {
                Console.WriteLine($"[info] Fallback link: {fallbackLink}");
            }
            return false;
        }
        catch (Exception ex)
        {
            Console.WriteLine($"[warn] Resend email send failed: {ex.Message}");
            if (!string.IsNullOrEmpty(fallbackLink))
            {
                Console.WriteLine($"[info] Fallback link: {fallbackLink}");
            }
            return false;
        }
    }

    private static string ExtractId(string body)
    {
        try
        {
            using var doc = JsonDocument.Parse(body);
            return doc.RootElement.TryGetProperty("id", out var id) ? id.GetString() ?? "?" : "?";
        }
        catch
        {
            return "?";
        }
    }

    private static string Mask(string email)
    {
        var at = email.IndexOf('@');
        if (at <= 1) return "***";
        return email[0] + "***" + email[at..];
    }

    private static string Truncate(string value, int max) =>
        string.IsNullOrEmpty(value) || value.Length <= max ? value : value[..max];
}
