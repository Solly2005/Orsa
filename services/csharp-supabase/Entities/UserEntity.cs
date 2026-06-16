namespace Orsa.SupabaseEngine.Entities;

public sealed class UserEntity
{
    public Guid Id { get; set; }
    public string Email { get; set; } = "";
    public string Username { get; set; } = "";

    /// <summary>PBKDF2-SHA256 hash for email/password users; null for OAuth users.</summary>
    public string? PasswordHash { get; set; }

    /// <summary>
    /// A password awaiting email-link confirmation before it becomes active. Used
    /// when someone sets a password on an existing Google account: the password is
    /// only promoted to <see cref="PasswordHash"/> once the verification link
    /// (which only the inbox owner can open) is clicked, preventing takeover.
    /// </summary>
    public string? PendingPasswordHash { get; set; }

    /// <summary>Google's stable subject identifier (sub claim). Null for email/password users.</summary>
    public string? GoogleSub { get; set; }

    /// <summary>"email", "google", or "both" once a Gmail account links a password.</summary>
    public string AuthProvider { get; set; } = "email";

    /// <summary>
    /// True once the email address is confirmed. Google sign-ins are verified by
    /// Google; email/password signups must click a Resend verification link.
    /// </summary>
    public bool EmailVerified { get; set; }

    public DateTime CreatedAtUtc { get; set; } = DateTime.UtcNow;
    public DateTime? LastLoginUtc { get; set; }
}
