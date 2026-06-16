using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;

namespace Orsa.SupabaseEngine.Services;

/// <summary>
/// Sends transactional email via the Mailtrap sending API. When MAILTRAP_API_TOKEN
/// is unset the service is a no-op (logs a warning) so local development works
/// without email delivery — the user simply lands unverified.
/// </summary>
public sealed class EmailService(IHttpClientFactory httpFactory, IConfiguration config)
{
    private const string MailtrapEndpoint = "https://send.api.mailtrap.io/api/send";

    public bool IsConfigured => !string.IsNullOrWhiteSpace(config["MAILTRAP_API_TOKEN"]);

    /// <summary>
    /// Sends the verification email. Returns true if the provider accepted it.
    /// Never throws: delivery failures are logged and reported as false so
    /// registration still succeeds (login is allowed; features stay gated).
    /// </summary>
    public async Task<bool> SendVerificationEmailAsync(string toEmail, string rawToken, CancellationToken ct)
    {
        var apiToken = config["MAILTRAP_API_TOKEN"]?.Trim();
        if (string.IsNullOrWhiteSpace(apiToken))
        {
            Console.WriteLine("[warn] MAILTRAP_API_TOKEN not set; skipping verification email. User remains unverified.");
            return false;
        }

        var fromAddress = config["MAILTRAP_FROM_ADDRESS"]?.Trim();
        var fromName = config["MAILTRAP_FROM_NAME"]?.Trim() ?? "ORSA";
        if (string.IsNullOrWhiteSpace(fromAddress))
        {
            Console.WriteLine("[warn] MAILTRAP_FROM_ADDRESS not set; skipping verification email.");
            return false;
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

        var text =
            "Welcome to ORSA.\n\n" +
            "Please confirm your email address to unlock chat and uploads:\n" +
            link + "\n\n" +
            "This link expires in 24 hours. If you did not create an ORSA account, you can ignore this email.";

        return await SendAsync(apiToken, fromAddress, fromName, toEmail, "Verify your ORSA email", html, text, link, ct);
    }

    private async Task<bool> SendAsync(
        string apiToken, string fromAddress, string fromName,
        string toEmail, string subject, string html, string text,
        string fallbackLink, CancellationToken ct)
    {
        var payload = JsonSerializer.Serialize(new
        {
            from = new { email = fromAddress, name = fromName },
            to = new[] { new { email = toEmail } },
            subject,
            html,
            text
        });

        try
        {
            using var http = httpFactory.CreateClient();
            using var req = new HttpRequestMessage(HttpMethod.Post, MailtrapEndpoint)
            {
                Content = new StringContent(payload, Encoding.UTF8, "application/json")
            };
            req.Headers.Authorization = new AuthenticationHeaderValue("Bearer", apiToken);

            using var resp = await http.SendAsync(req, ct);
            var body = await resp.Content.ReadAsStringAsync(ct);
            if (resp.IsSuccessStatusCode)
            {
                Console.WriteLine($"[info] Mailtrap accepted email to {Mask(toEmail)}.");
                return true;
            }

            Console.WriteLine($"[warn] Mailtrap rejected email to {Mask(toEmail)} ({(int)resp.StatusCode}): {Truncate(body, 400)}");
            Console.WriteLine("[warn] Check MAILTRAP_FROM_ADDRESS uses a verified domain and MAILTRAP_API_TOKEN is a sending token.");
            if (!string.IsNullOrEmpty(fallbackLink))
                Console.WriteLine($"[info] Fallback link: {fallbackLink}");
            return false;
        }
        catch (Exception ex)
        {
            Console.WriteLine($"[warn] Mailtrap email send failed: {ex.Message}");
            if (!string.IsNullOrEmpty(fallbackLink))
                Console.WriteLine($"[info] Fallback link: {fallbackLink}");
            return false;
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
