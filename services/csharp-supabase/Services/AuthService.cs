using System.Net.Http.Headers;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using Microsoft.EntityFrameworkCore;
using Orsa.SupabaseEngine.Data;
using Orsa.SupabaseEngine.Entities;

namespace Orsa.SupabaseEngine.Services;

// ── Request records ──────────────────────────────────────────────────────────

public sealed record RegisterRequest(
    string Email,
    string Password,
    string? AcceptedLegalVersion,
    bool MemoryExtractionEnabled);

public sealed record LoginRequest(string Email, string Password);

public sealed record GoogleExchangeRequest(string Code, string? State);

// ── Service ──────────────────────────────────────────────────────────────────

/// <summary>
/// HTTP auth handlers: register, login, Google OAuth redirect + code exchange.
/// Registered as a scoped service; endpoints are mapped in Program.cs.
/// </summary>
public sealed class AuthService(OrsaDbContext db, IHttpClientFactory httpFactory, IConfiguration config)
{
    private const int MinPasswordLength = 8;
    private const int MaxPasswordLength = 200;

    // ── Register ─────────────────────────────────────────────────────────────

    public async Task<IResult> RegisterAsync(RegisterRequest req, CancellationToken ct)
    {
        if (string.IsNullOrWhiteSpace(req.Email) || string.IsNullOrWhiteSpace(req.Password))
            return Results.BadRequest(new { error = "email and password are required" });

        if (!IsAcceptablePassword(req.Password))
            return Results.BadRequest(new { error = $"password must be between {MinPasswordLength} and {MaxPasswordLength} characters" });

        var email = req.Email.Trim().ToLowerInvariant();

        // Generic conflict message: do not confirm which email is registered.
        // (Eliminating enumeration entirely would require email verification.)
        if (await db.Users.AnyAsync(u => u.Email == email, ct))
            return Results.Conflict(new { error = "unable to register with the provided details" });

        var userId = Guid.NewGuid();
        var user = new UserEntity
        {
            Id = userId,
            Email = email,
            Username = email.Split('@')[0],
            PasswordHash = HashPassword(req.Password),
            AuthProvider = "email",
            CreatedAtUtc = DateTime.UtcNow
        };
        db.Users.Add(user);

        if (!string.IsNullOrWhiteSpace(req.AcceptedLegalVersion))
        {
            db.LegalAcceptances.Add(new LegalAcceptanceEntity
            {
                UserId = userId,
                TermsVersion = req.AcceptedLegalVersion,
                PrivacyVersion = req.AcceptedLegalVersion,
                ConsentVersion = req.AcceptedLegalVersion,
                AcceptedAtUtc = DateTime.UtcNow
            });
        }

        await db.SaveChangesAsync(ct);

        return Results.Ok(new
        {
            userId = userId.ToString(),
            email = user.Email,
            token = IssueToken(userId, user.Email),
            acceptedLegalVersion = req.AcceptedLegalVersion,
            memoryExtractionEnabled = req.MemoryExtractionEnabled,
            sessionRestored = true
        });
    }

    // ── Login ────────────────────────────────────────────────────────────────

    public async Task<IResult> LoginAsync(LoginRequest req, CancellationToken ct)
    {
        if (string.IsNullOrWhiteSpace(req.Email) || string.IsNullOrWhiteSpace(req.Password))
            return Results.BadRequest(new { error = "email and password are required" });

        var email = req.Email.Trim().ToLowerInvariant();
        var user = await db.Users
            .FirstOrDefaultAsync(u => u.Email == email && u.AuthProvider == "email", ct);

        // Use constant-time comparison to avoid timing attacks; never reveal which field was wrong.
        if (user is null || string.IsNullOrEmpty(user.PasswordHash) ||
            !VerifyPassword(req.Password, user.PasswordHash))
        {
            return Results.Unauthorized();
        }

        // Transparently upgrade hashes created with a weaker work factor.
        if (NeedsRehash(user.PasswordHash))
        {
            user.PasswordHash = HashPassword(req.Password);
        }

        user.LastLoginUtc = DateTime.UtcNow;
        await db.SaveChangesAsync(ct);

        return Results.Ok(new
        {
            userId = user.Id.ToString(),
            email = user.Email,
            token = IssueToken(user.Id, user.Email),
            persistent = true
        });
    }

    // ── Google — redirect ────────────────────────────────────────────────────

    public static IResult GoogleRedirect(HttpContext ctx, IConfiguration config)
    {
        var clientId = config["GOOGLE_CLIENT_ID"]?.Trim();
        var redirectUri = config["GOOGLE_REDIRECT_URI"]?.Trim();

        if (string.IsNullOrEmpty(clientId) || string.IsNullOrEmpty(redirectUri))
            return Results.Problem(
                detail: "Set GOOGLE_CLIENT_ID and GOOGLE_REDIRECT_URI environment variables to enable Google sign-in.",
                title: "Google OAuth is not configured",
                statusCode: 503);

        var state = Convert.ToHexString(RandomNumberGenerator.GetBytes(16)).ToLowerInvariant();
        ctx.Response.Cookies.Append("orsa-oauth-state", state, new CookieOptions
        {
            HttpOnly = true,
            SameSite = SameSiteMode.Lax,
            MaxAge = TimeSpan.FromMinutes(5),
            Path = "/"
        });

        var qs = QueryString.Create(new Dictionary<string, string?>
        {
            ["client_id"] = clientId,
            ["redirect_uri"] = redirectUri,
            ["response_type"] = "code",
            ["scope"] = "openid email profile",
            ["state"] = state,
            ["access_type"] = "offline",
            ["prompt"] = "select_account"
        });

        return Results.Redirect($"https://accounts.google.com/o/oauth2/v2/auth{qs}");
    }

    // ── Google — code exchange ───────────────────────────────────────────────

    public async Task<IResult> GoogleExchangeAsync(
        GoogleExchangeRequest req, HttpContext httpCtx, IConfiguration config, CancellationToken ct)
    {
        var clientId = config["GOOGLE_CLIENT_ID"]?.Trim();
        var clientSecret = config["GOOGLE_CLIENT_SECRET"]?.Trim();
        var redirectUri = config["GOOGLE_REDIRECT_URI"]?.Trim();

        if (string.IsNullOrEmpty(clientId) || string.IsNullOrEmpty(clientSecret) || string.IsNullOrEmpty(redirectUri))
            return Results.Problem(
                detail: "Set GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, and GOOGLE_REDIRECT_URI to enable Google sign-in.",
                title: "Google OAuth is not configured",
                statusCode: 503);

        if (string.IsNullOrWhiteSpace(req.Code))
            return Results.BadRequest(new { error = "authorization code is required" });

        // CSRF protection: the state echoed back by Google must match the
        // single-use cookie set during GoogleRedirect. Without this check an
        // attacker could inject their own authorization code (login CSRF).
        var stateCookie = httpCtx.Request.Cookies["orsa-oauth-state"];
        httpCtx.Response.Cookies.Delete("orsa-oauth-state");
        if (string.IsNullOrEmpty(stateCookie) || !FixedTimeEquals(stateCookie, req.State))
            return Results.BadRequest(new { error = "invalid or expired sign-in state; please try again" });

        using var http = httpFactory.CreateClient();

        // Exchange authorization code for access token.
        var tokenResp = await http.PostAsync("https://oauth2.googleapis.com/token",
            new FormUrlEncodedContent(new Dictionary<string, string>
            {
                ["code"] = req.Code,
                ["client_id"] = clientId,
                ["client_secret"] = clientSecret,
                ["redirect_uri"] = redirectUri,
                ["grant_type"] = "authorization_code"
            }), ct);

        var tokenJson = await tokenResp.Content.ReadAsStringAsync(ct);
        using var tokenDoc = JsonDocument.Parse(tokenJson);

        if (!tokenDoc.RootElement.TryGetProperty("access_token", out var accessTokenEl))
        {
            var errMsg = tokenDoc.RootElement.TryGetProperty("error", out var errEl)
                ? errEl.GetString() : "token exchange failed";
            return Results.Problem(detail: errMsg, statusCode: 502);
        }

        // Fetch Google user info.
        using var userInfoReq = new HttpRequestMessage(HttpMethod.Get,
            "https://www.googleapis.com/oauth2/v3/userinfo");
        userInfoReq.Headers.Authorization =
            new AuthenticationHeaderValue("Bearer", accessTokenEl.GetString());
        var userInfoResp = await http.SendAsync(userInfoReq, ct);

        var userInfoJson = await userInfoResp.Content.ReadAsStringAsync(ct);
        using var userInfoDoc = JsonDocument.Parse(userInfoJson);
        var root = userInfoDoc.RootElement;

        if (!root.TryGetProperty("email", out var emailEl))
            return Results.Problem(detail: "could not read user info from Google", statusCode: 502);

        var email = emailEl.GetString()!.Trim().ToLowerInvariant();
        var googleSub = root.TryGetProperty("sub", out var subEl) ? subEl.GetString() ?? "" : "";
        var displayName = root.TryGetProperty("name", out var nameEl) ? nameEl.GetString() ?? "" : "";

        // Find existing user by Google subject id, or fall back to email.
        var user = await db.Users.FirstOrDefaultAsync(
            u => u.GoogleSub == googleSub || (u.Email == email && u.AuthProvider == "google"), ct);

        if (user is null)
        {
            user = new UserEntity
            {
                Id = Guid.NewGuid(),
                Email = email,
                Username = string.IsNullOrWhiteSpace(displayName) ? email.Split('@')[0] : displayName,
                GoogleSub = googleSub,
                AuthProvider = "google",
                CreatedAtUtc = DateTime.UtcNow,
                LastLoginUtc = DateTime.UtcNow
            };
            db.Users.Add(user);
        }
        else
        {
            user.LastLoginUtc = DateTime.UtcNow;
            if (!string.IsNullOrEmpty(googleSub)) user.GoogleSub = googleSub;
        }

        await db.SaveChangesAsync(ct);

        return Results.Ok(new
        {
            userId = user.Id.ToString(),
            email = user.Email,
            token = IssueToken(user.Id, user.Email),
            displayName,
            provider = "google",
            persistent = true
        });
    }

    // ── Token + validation helpers ───────────────────────────────────────────

    private string IssueToken(Guid userId, string email) =>
        SessionTokens.Issue(SessionTokens.ResolveSecret(config), userId, email);

    private static bool IsAcceptablePassword(string password) =>
        password.Length >= MinPasswordLength && password.Length <= MaxPasswordLength;

    private static bool FixedTimeEquals(string? a, string? b)
    {
        if (a is null || b is null) return false;
        var ba = Encoding.UTF8.GetBytes(a);
        var bb = Encoding.UTF8.GetBytes(b);
        return ba.Length == bb.Length && CryptographicOperations.FixedTimeEquals(ba, bb);
    }

    // ── Password helpers ─────────────────────────────────────────────────────

    // OWASP-aligned PBKDF2-SHA256 work factor. The hash is self-describing, so
    // existing lower-iteration hashes still verify and are transparently
    // upgraded on next successful login (see LoginAsync rehash).
    private const int Pbkdf2Iterations = 600_000;

    /// <summary>
    /// Produces a self-describing PBKDF2-SHA256 hash:
    /// <c>pbkdf2-sha256:{iterations}:{saltBase64}:{hashBase64}</c>
    /// </summary>
    private static string HashPassword(string password)
    {
        var salt = RandomNumberGenerator.GetBytes(32);
        var hash = Rfc2898DeriveBytes.Pbkdf2(
            Encoding.UTF8.GetBytes(password), salt, Pbkdf2Iterations, HashAlgorithmName.SHA256, 32);
        return $"pbkdf2-sha256:{Pbkdf2Iterations}:{Convert.ToBase64String(salt)}:{Convert.ToBase64String(hash)}";
    }

    private static bool VerifyPassword(string password, string storedHash)
    {
        var parts = storedHash.Split(':');
        if (parts.Length != 4 || parts[0] != "pbkdf2-sha256") return false;
        if (!int.TryParse(parts[1], out var iterations)) return false;
        var salt = Convert.FromBase64String(parts[2]);
        var expected = Convert.FromBase64String(parts[3]);
        var actual = Rfc2898DeriveBytes.Pbkdf2(
            Encoding.UTF8.GetBytes(password), salt, iterations, HashAlgorithmName.SHA256, 32);
        return CryptographicOperations.FixedTimeEquals(actual, expected);
    }

    private static bool NeedsRehash(string storedHash)
    {
        var parts = storedHash.Split(':');
        return parts.Length != 4
            || parts[0] != "pbkdf2-sha256"
            || !int.TryParse(parts[1], out var iterations)
            || iterations < Pbkdf2Iterations;
    }
}
