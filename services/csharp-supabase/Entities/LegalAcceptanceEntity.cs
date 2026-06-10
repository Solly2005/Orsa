namespace Orsa.SupabaseEngine.Entities;

public sealed class LegalAcceptanceEntity
{
    public Guid Id { get; set; } = Guid.NewGuid();
    public Guid UserId { get; set; }
    public string TermsVersion { get; set; } = "";
    public string PrivacyVersion { get; set; } = "";
    public string ConsentVersion { get; set; } = "";
    public DateTime AcceptedAtUtc { get; set; }
}
