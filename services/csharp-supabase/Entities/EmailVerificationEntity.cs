namespace Orsa.SupabaseEngine.Entities;

/// <summary>
/// A pending email-verification challenge. Only a SHA-256 hash of the token is
/// stored; the raw token travels in the emailed link. Rows are single-use
/// (ConsumedAtUtc) and time-boxed (ExpiresAtUtc).
/// </summary>
public sealed class EmailVerificationEntity
{
    public Guid Id { get; set; }
    public Guid UserId { get; set; }
    public string TokenHash { get; set; } = "";
    public DateTime ExpiresAtUtc { get; set; }
    public DateTime? ConsumedAtUtc { get; set; }
    public DateTime CreatedAtUtc { get; set; } = DateTime.UtcNow;
}
