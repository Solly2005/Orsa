namespace Orsa.SupabaseEngine.Entities;

public sealed class UserEntity
{
    public Guid Id { get; set; }
    public string Email { get; set; } = "";
    public string Username { get; set; } = "";

    /// <summary>PBKDF2-SHA256 hash for email/password users; null for OAuth users.</summary>
    public string? PasswordHash { get; set; }

    /// <summary>Google's stable subject identifier (sub claim). Null for email/password users.</summary>
    public string? GoogleSub { get; set; }

    /// <summary>"email" or "google". Defaults to "email".</summary>
    public string AuthProvider { get; set; } = "email";

    public DateTime CreatedAtUtc { get; set; } = DateTime.UtcNow;
    public DateTime? LastLoginUtc { get; set; }
}
