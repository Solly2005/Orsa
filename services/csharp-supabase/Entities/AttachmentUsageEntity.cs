namespace Orsa.SupabaseEngine.Entities;

/// <summary>
/// Authoritative per-user, per-day attachment-upload counter. Replaces the Go
/// gateway's in-memory counter so the quota survives restarts and is shared
/// across instances. Keyed by (UserId, UsageDate in UTC).
/// </summary>
public sealed class AttachmentUsageEntity
{
    public Guid UserId { get; set; }
    public DateOnly UsageDate { get; set; }
    public int Count { get; set; }
}
