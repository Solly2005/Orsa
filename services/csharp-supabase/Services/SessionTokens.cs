using System.Security.Cryptography;
using System.Text;
using System.Text.Json;

namespace Orsa.SupabaseEngine.Services;

/// <summary>
/// Mints HS256 session tokens (compact JWTs) that the Go gateway verifies on
/// every REST call. Both services share the secret via ORSA_SESSION_SECRET so
/// the gateway can verify locally without calling back here. Kept dependency-free
/// and symmetric with the Go service's internal/auth package.
/// </summary>
public static class SessionTokens
{
    /// <summary>Session lifetime; matches the "stay signed in" UX.</summary>
    public static readonly TimeSpan Ttl = TimeSpan.FromDays(7);

    // Insecure fallback used when ORSA_SESSION_SECRET is unset, matching the Go
    // service's auth.DevSecret. Never use in production.
    public const string DevSecret = "orsa-dev-insecure-session-secret-change-me";

    public static string ResolveSecret(IConfiguration config)
    {
        var configured = (config["ORSA_SESSION_SECRET"] ?? config["SESSION_SECRET"])?.Trim();
        return string.IsNullOrWhiteSpace(configured) ? DevSecret : configured;
    }

    public static string Issue(string secret, Guid userId, string email, bool emailVerified)
    {
        var now = DateTimeOffset.UtcNow;
        var header = Base64Url(JsonSerializer.SerializeToUtf8Bytes(new { alg = "HS256", typ = "JWT" }));
        var payload = Base64Url(JsonSerializer.SerializeToUtf8Bytes(new
        {
            sub = userId.ToString(),
            email = email ?? string.Empty,
            email_verified = emailVerified,
            iat = now.ToUnixTimeSeconds(),
            exp = now.Add(Ttl).ToUnixTimeSeconds()
        }));
        var signingInput = $"{header}.{payload}";
        return $"{signingInput}.{Base64Url(HmacSha256(secret, signingInput))}";
    }

    private static byte[] HmacSha256(string secret, string input)
    {
        using var mac = new HMACSHA256(Encoding.UTF8.GetBytes(secret));
        return mac.ComputeHash(Encoding.UTF8.GetBytes(input));
    }

    private static string Base64Url(byte[] bytes) =>
        Convert.ToBase64String(bytes).TrimEnd('=').Replace('+', '-').Replace('/', '_');
}
