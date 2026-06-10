namespace Orsa.SupabaseEngine.Entities;

public sealed class UserSettingsEntity
{
    public Guid UserId { get; set; }
    public string ThemePreference { get; set; } = "system";
    public string HealthProfileData { get; set; } = "{}";
    public DateTime UpdatedAtUtc { get; set; } = DateTime.UtcNow;
}
