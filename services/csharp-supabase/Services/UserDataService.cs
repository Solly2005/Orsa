using System.Globalization;
using System.Text.Json;
using System.Text.Json.Nodes;
using Microsoft.EntityFrameworkCore;
using Orsa.SupabaseEngine.Data;
using Orsa.SupabaseEngine.Entities;

namespace Orsa.SupabaseEngine.Services;

public sealed record UserSettingsView(
    string ApiVersion,
    string UserId,
    bool MemoryExtractionEnabled,
    bool RemindersEnabled,
    int AttachmentCountToday,
    int AttachmentLimit);

public sealed record ProfileView(
    string ApiVersion,
    string UserId,
    string DisplayName,
    string Country,
    string Region,
    string City,
    string PersonaJson,
    string PersonaUpdatedAt,
    string PersonaSummary,
    string WorkflowBoundary,
    string ConsentStatus,
    string BoundaryPrompt);

public sealed record WriteAckView(string ApiVersion, bool Ok, string Id);

public sealed class UserDataService(OrsaDbContext db)
{
    private const string DefaultPersonaSummary =
        "Persona extraction is stored separately from triage and only runs with explicit consent.";

    private const string DefaultWorkflowBoundary =
        "Stored profile context can personalize response style only when consent is enabled. It must not change clinical urgency, diagnosis, or safety escalation.";

    public async Task<UserSettingsView> GetSettingsAsync(Guid userId, CancellationToken cancellationToken)
    {
        var settings = await GetOrCreateSettings(userId, cancellationToken);
        return ToSettingsView(settings);
    }

    public async Task<UserSettingsView> UpdateSettingsAsync(Guid userId, bool? memoryExtractionEnabled, bool? remindersEnabled, CancellationToken cancellationToken)
    {
        var settings = await GetOrCreateSettings(userId, cancellationToken);
        var data = ReadHealthProfileData(settings);

        if (memoryExtractionEnabled.HasValue)
        {
            data["memoryExtractionEnabled"] = memoryExtractionEnabled.Value;
        }

        if (remindersEnabled.HasValue)
        {
            data["remindersEnabled"] = remindersEnabled.Value;
        }

        settings.HealthProfileData = data.ToJsonString();
        settings.UpdatedAtUtc = DateTime.UtcNow;
        await db.SaveChangesAsync(cancellationToken);
        return ToSettingsView(settings);
    }

    public async Task<ProfileView> GetProfileAsync(Guid userId, CancellationToken cancellationToken)
    {
        var user = await db.Users.FirstOrDefaultAsync(x => x.Id == userId, cancellationToken);
        var settings = await GetOrCreateSettings(userId, cancellationToken);
        var data = ReadHealthProfileData(settings);
        return BuildProfileView(userId, user, data);
    }

    public async Task<ProfileView> UpdateProfileAsync(
        Guid userId,
        bool? memoryExtractionEnabled,
        string? personaSummary,
        string? workflowBoundary,
        CancellationToken cancellationToken)
    {
        var settings = await GetOrCreateSettings(userId, cancellationToken);
        var data = ReadHealthProfileData(settings);

        if (memoryExtractionEnabled.HasValue)
        {
            data["memoryExtractionEnabled"] = memoryExtractionEnabled.Value;
        }

        if (personaSummary is not null)
        {
            data["personaSummary"] = TrimProfileText(personaSummary, 2000);
        }

        if (workflowBoundary is not null)
        {
            data["workflowBoundary"] = TrimProfileText(workflowBoundary, 1600);
        }

        var consentEnabled = ReadBool(data, "memoryExtractionEnabled", false);
        var summary = ReadString(data, "personaSummary", DefaultPersonaSummary);
        var boundary = ReadString(data, "workflowBoundary", DefaultWorkflowBoundary);
        data["boundaryPrompt"] = BuildBoundaryPrompt(consentEnabled, summary, boundary);

        settings.HealthProfileData = data.ToJsonString();
        settings.UpdatedAtUtc = DateTime.UtcNow;
        await db.SaveChangesAsync(cancellationToken);

        var user = await db.Users.FirstOrDefaultAsync(x => x.Id == userId, cancellationToken);
        return BuildProfileView(userId, user, data);
    }

    public async Task<WriteAckView> RecordLegalAcceptanceAsync(
        Guid userId,
        string termsVersion,
        string privacyVersion,
        string consentVersion,
        string acceptedAtIso,
        CancellationToken cancellationToken)
    {
        var entity = new LegalAcceptanceEntity
        {
            UserId = userId,
            TermsVersion = termsVersion,
            PrivacyVersion = privacyVersion,
            ConsentVersion = consentVersion,
            AcceptedAtUtc = ParseIsoOrUtcNow(acceptedAtIso)
        };

        db.LegalAcceptances.Add(entity);
        await db.SaveChangesAsync(cancellationToken);
        return Ack(entity.Id);
    }

    public async Task<WriteAckView> WritePersonaAuditAsync(
        Guid userId,
        IReadOnlyCollection<string> sourceThreadIds,
        string promptHash,
        string modelId,
        string status,
        string extractedJson,
        string error,
        string runAtIso,
        CancellationToken cancellationToken)
    {
        var entity = new PersonaAuditEntity
        {
            UserId = userId,
            SourceThreadIdsJson = JsonSerializer.Serialize(sourceThreadIds.ToArray()),
            PromptHash = promptHash,
            ModelId = modelId,
            Status = status,
            ExtractedJson = NormalizeJsonOrEmpty(extractedJson),
            Error = error,
            RunAtUtc = ParseIsoOrUtcNow(runAtIso)
        };

        db.PersonaAudits.Add(entity);

        if (status == "succeeded" && !string.IsNullOrWhiteSpace(extractedJson))
        {
            var settings = await GetOrCreateSettings(userId, cancellationToken);
            var data = ReadHealthProfileData(settings);
            data["persona"] = ParseJsonNodeOrString(extractedJson);
            data["personaUpdatedAt"] = entity.RunAtUtc.ToString("O", CultureInfo.InvariantCulture);
            data["personaPromptHash"] = promptHash;
            data["personaModelId"] = modelId;
            settings.HealthProfileData = data.ToJsonString();
            settings.UpdatedAtUtc = DateTime.UtcNow;
        }

        await db.SaveChangesAsync(cancellationToken);
        return Ack(entity.Id);
    }

    private async Task<UserSettingsEntity> GetOrCreateSettings(Guid userId, CancellationToken cancellationToken)
    {
        var settings = await db.UserSettings.FirstOrDefaultAsync(x => x.UserId == userId, cancellationToken);
        if (settings is not null)
        {
            return settings;
        }

        var userExists = await db.Users.AnyAsync(x => x.Id == userId, cancellationToken);
        if (!userExists)
        {
            db.Users.Add(new UserEntity
            {
                Id = userId,
                Email = "",
                Username = "",
                CreatedAtUtc = DateTime.UtcNow
            });
        }

        settings = new UserSettingsEntity
        {
            UserId = userId,
            ThemePreference = "system",
            HealthProfileData = new JsonObject
            {
                ["memoryExtractionEnabled"] = false,
                ["remindersEnabled"] = true,
                ["personaSummary"] = DefaultPersonaSummary,
                ["workflowBoundary"] = DefaultWorkflowBoundary
            }.ToJsonString(),
            UpdatedAtUtc = DateTime.UtcNow
        };
        db.UserSettings.Add(settings);
        await db.SaveChangesAsync(cancellationToken);
        return settings;
    }

    private static UserSettingsView ToSettingsView(UserSettingsEntity settings) => new(
        "v1",
        settings.UserId.ToString(),
        ReadBool(settings.HealthProfileData, "memoryExtractionEnabled", false),
        ReadBool(settings.HealthProfileData, "remindersEnabled", true),
        0,
        5);

    private static ProfileView BuildProfileView(Guid userId, UserEntity? user, JsonObject data)
    {
        var consentEnabled = ReadBool(data, "memoryExtractionEnabled", false);
        var summary = ReadString(data, "personaSummary", DefaultPersonaSummary);
        var boundary = ReadString(data, "workflowBoundary", DefaultWorkflowBoundary);
        var prompt = BuildBoundaryPrompt(consentEnabled, summary, boundary);

        return new ProfileView(
            "v1",
            userId.ToString(),
            user?.Username ?? user?.Email ?? "ORSA User",
            "",
            "",
            "",
            data.ToJsonString(),
            ReadString(data, "personaUpdatedAt", ""),
            summary,
            boundary,
            consentEnabled ? "enabled" : "disabled",
            prompt);
    }

    private static WriteAckView Ack(Guid id) => new("v1", true, id.ToString());

    private static DateTime ParseIsoOrUtcNow(string value)
    {
        return DateTime.TryParse(value, CultureInfo.InvariantCulture, DateTimeStyles.AdjustToUniversal, out var parsed)
            ? parsed.ToUniversalTime()
            : DateTime.UtcNow;
    }

    private static JsonObject ReadHealthProfileData(UserSettingsEntity settings)
    {
        return JsonNode.Parse(string.IsNullOrWhiteSpace(settings.HealthProfileData) ? "{}" : settings.HealthProfileData) as JsonObject
            ?? new JsonObject();
    }

    private static bool ReadBool(string json, string key, bool fallback)
    {
        var data = JsonNode.Parse(string.IsNullOrWhiteSpace(json) ? "{}" : json) as JsonObject;
        return data is null ? fallback : ReadBool(data, key, fallback);
    }

    private static bool ReadBool(JsonObject data, string key, bool fallback)
    {
        try
        {
            return data[key]?.GetValue<bool>() ?? fallback;
        }
        catch (InvalidOperationException)
        {
            return fallback;
        }
    }

    private static string ReadString(JsonObject data, string key, string fallback)
    {
        try
        {
            return data[key]?.GetValue<string>() ?? fallback;
        }
        catch (InvalidOperationException)
        {
            return fallback;
        }
    }

    private static string TrimProfileText(string value, int maxLength)
    {
        var trimmed = (value ?? string.Empty).Trim();
        return trimmed.Length <= maxLength ? trimmed : trimmed[..maxLength];
    }

    private static string BuildBoundaryPrompt(bool consentEnabled, string personaSummary, string workflowBoundary)
    {
        if (!consentEnabled)
        {
            return "Personalization consent is disabled. Do not use stored persona summary or workflow boundary in this thread.";
        }

        var parts = new List<string>
        {
            "User-approved profile context is available for GPT-OSS in this thread.",
            "Use it only to respect communication preferences and workflow boundaries."
        };

        if (!string.IsNullOrWhiteSpace(personaSummary))
        {
            parts.Add("Persona summary: " + personaSummary);
        }

        if (!string.IsNullOrWhiteSpace(workflowBoundary))
        {
            parts.Add("Workflow boundary: " + workflowBoundary);
        }

        parts.Add("This context is not clinical evidence. Do not infer symptoms, history, risk, diagnoses, or severity from it. Never let it reduce urgency, override safety rules, or bypass escalation.");
        return string.Join(" ", parts);
    }

    private static JsonNode ParseJsonNodeOrString(string value)
    {
        try
        {
            return JsonNode.Parse(value) ?? JsonValue.Create(value)!;
        }
        catch (JsonException)
        {
            return JsonValue.Create(value)!;
        }
    }

    private static string NormalizeJsonOrEmpty(string value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return "{}";
        }

        try
        {
            return JsonNode.Parse(value)?.ToJsonString() ?? "{}";
        }
        catch (JsonException)
        {
            return JsonSerializer.Serialize(new { value });
        }
    }
}
