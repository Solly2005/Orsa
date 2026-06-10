namespace Orsa.SupabaseEngine.Entities;

public sealed class PersonaAuditEntity
{
    public Guid Id { get; set; } = Guid.NewGuid();
    public Guid UserId { get; set; }
    public string SourceThreadIdsJson { get; set; } = "[]";
    public string PromptHash { get; set; } = "";
    public string ModelId { get; set; } = "";
    public string Status { get; set; } = "";
    public string ExtractedJson { get; set; } = "";
    public string Error { get; set; } = "";
    public DateTime RunAtUtc { get; set; }
}
