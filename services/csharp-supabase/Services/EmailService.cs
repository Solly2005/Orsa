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
            $"<p>Welcome to ORSA.</p>" +
            $"<p>Please confirm your email address to unlock chat and uploads:</p>" +
            $"<p><a href=\"{link}\">Verify my email</a></p>" +
            $"<p>If the button does not work, paste this link into your browser:<br>{link}</p>" +
            $"<p>This link expires in 24 hours. If you did not create an ORSA account, you can ignore this email.</p>";

        var payload = JsonSerializer.Serialize(new
        {
            from,
            to = new[] { toEmail },
            subject = "Verify your ORSA email",
            html
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
            if (resp.IsSuccessStatusCode)
            {
                return true;
            }

            var body = await resp.Content.ReadAsStringAsync(ct);
            Console.WriteLine($"[warn] Resend rejected verification email ({(int)resp.StatusCode}): {Truncate(body, 300)}");
            Console.WriteLine($"[info] Fallback verification link: {link}");
            return false;
        }
        catch (Exception ex)
        {
            Console.WriteLine($"[warn] Resend verification email failed: {ex.Message}");
            Console.WriteLine($"[info] Fallback verification link: {link}");
            return false;
        }
    }

    private static string Truncate(string value, int max) =>
        string.IsNullOrEmpty(value) || value.Length <= max ? value : value[..max];
}
