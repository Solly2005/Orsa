using System.Globalization;
using System.Text.Json;
using System.Text.Json.Nodes;
using Grpc.Core;
using Microsoft.EntityFrameworkCore;
using Orsa.Proto.User.V1;
using Orsa.SupabaseEngine.Data;
using Orsa.SupabaseEngine.Entities;

namespace Orsa.SupabaseEngine.Services;

public sealed class UserGrpcService(OrsaDbContext db) : UserService.UserServiceBase
{
    private const string DefaultPersonaSummary =
        "Persona extraction is stored separately from triage and only runs with explicit consent.";

    private const string DefaultWorkflowBoundary =
        "Stored profile context can personalize response style only when consent is enabled. It must not change clinical urgency, diagnosis, or safety escalation.";

    public override async Task<UserSettings> GetSettings(UserIdRequest request, ServerCallContext context)
    {
        var userId = ParseUserId(request.UserId);
        var settings = await GetOrCreateSettings(userId, context.CancellationToken);
        return ToProto(settings);
    }

    public override async Task<UserSettings> UpdateSettings(UpdateSettingsRequest request, ServerCallContext context)
    {
        var userId = ParseUserId(request.UserId);
        var settings = await GetOrCreateSettings(userId, context.CancellationToken);
        var data = ReadHealthProfileData(settings);

        if (request.HasMemoryExtractionEnabled)
        {
            data["memoryExtractionEnabled"] = request.MemoryExtractionEnabled;
        }

        if (request.HasRemindersEnabled)
        {
            data["remindersEnabled"] = request.RemindersEnabled;
        }

        settings.HealthProfileData = data.ToJsonString();
        settings.UpdatedAtUtc = DateTime.UtcNow;
        await db.SaveChangesAsync(context.CancellationToken);
        return ToProto(settings);
    }

    public override async Task<ProfileResponse> GetProfile(UserIdRequest request, ServerCallContext context)
    {
        var userId = ParseUserId(request.UserId);
        var user = await db.Users.FirstOrDefaultAsync(x => x.Id == userId, context.CancellationToken);
        var settings = await GetOrCreateSettings(userId, context.CancellationToken);
        var data = ReadHealthProfileData(settings);

        return BuildProfileResponse(userId, user, data);
    }

    public override async Task<ProfileResponse> UpdateProfile(UpdateProfileRequest request, ServerCallContext context)
    {
        var userId = ParseUserId(request.UserId);
        var settings = await GetOrCreateSettings(userId, context.CancellationToken);
        var data = ReadHealthProfileData(settings);

        if (request.HasMemoryExtractionEnabled)
        {
            data["memoryExtractionEnabled"] = request.MemoryExtractionEnabled;
        }

        if (request.HasPersonaSummary)
        {
            data["personaSummary"] = TrimProfileText(request.PersonaSummary, 2000);
        }

        if (request.HasWorkflowBoundary)
        {
            data["workflowBoundary"] = TrimProfileText(request.WorkflowBoundary, 1600);
        }

        var consentEnabled = ReadBool(data, "memoryExtractionEnabled", false);
        var summary = ReadString(data, "personaSummary", DefaultPersonaSummary);
        var boundary = ReadString(data, "workflowBoundary", DefaultWorkflowBoundary);
        data["boundaryPrompt"] = BuildBoundaryPrompt(consentEnabled, summary, boundary);

        settings.HealthProfileData = data.ToJsonString();
        settings.UpdatedAtUtc = DateTime.UtcNow;
        await db.SaveChangesAsync(context.CancellationToken);

        var user = await db.Users.FirstOrDefaultAsync(x => x.Id == userId, context.CancellationToken);
        return BuildProfileResponse(userId, user, data);
    }

    public override async Task<WriteAck> RecordLegalAcceptance(LegalAcceptanceRequest request, ServerCallContext context)
    {
        var userId = ParseUserId(request.UserId);
        var acceptedAt = ParseIsoOrUtcNow(request.AcceptedAtIso);
        var entity = new LegalAcceptanceEntity
        {
            UserId = userId,
            TermsVersion = request.TermsVersion,
            PrivacyVersion = request.PrivacyVersion,
            ConsentVersion = request.ConsentVersion,
            AcceptedAtUtc = acceptedAt
        };

        db.LegalAcceptances.Add(entity);
        await db.SaveChangesAsync(context.CancellationToken);
        return Ack(entity.Id);
    }

    public override async Task<WriteAck> WritePersonaAudit(PersonaAuditRequest request, ServerCallContext context)
    {
        var userId = ParseUserId(request.UserId);
        var entity = new PersonaAuditEntity
        {
            UserId = userId,
            SourceThreadIdsJson = JsonSerializer.Serialize(request.SourceThreadIds.ToArray()),
            PromptHash = request.PromptHash,
            ModelId = request.ModelId,
            Status = request.Status,
            ExtractedJson = NormalizeJsonOrEmpty(request.ExtractedJson),
            Error = request.Error,
            RunAtUtc = ParseIsoOrUtcNow(request.RunAtIso)
        };

        db.PersonaAudits.Add(entity);

        if (request.Status == "succeeded" && !string.IsNullOrWhiteSpace(request.ExtractedJson))
        {
            var settings = await GetOrCreateSettings(userId, context.CancellationToken);
            var data = ReadHealthProfileData(settings);
            data["persona"] = ParseJsonNodeOrString(request.ExtractedJson);
            data["personaUpdatedAt"] = entity.RunAtUtc.ToString("O", CultureInfo.InvariantCulture);
            data["personaPromptHash"] = request.PromptHash;
            data["personaModelId"] = request.ModelId;
            settings.HealthProfileData = data.ToJsonString();
            settings.UpdatedAtUtc = DateTime.UtcNow;
        }

        await db.SaveChangesAsync(context.CancellationToken);
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

    private static UserSettings ToProto(UserSettingsEntity settings) => new()
    {
        ApiVersion = "v1",
        UserId = settings.UserId.ToString(),
        MemoryExtractionEnabled = ReadBool(settings.HealthProfileData, "memoryExtractionEnabled", false),
        RemindersEnabled = ReadBool(settings.HealthProfileData, "remindersEnabled", true),
        AttachmentCountToday = 0,
        AttachmentLimit = 5
    };

    private static WriteAck Ack(Guid id) => new()
    {
        ApiVersion = "v1",
        Ok = true,
        Id = id.ToString()
    };

    private static ProfileResponse BuildProfileResponse(Guid userId, UserEntity? user, JsonObject data)
    {
        var consentEnabled = ReadBool(data, "memoryExtractionEnabled", false);
        var summary = ReadString(data, "personaSummary", DefaultPersonaSummary);
        var boundary = ReadString(data, "workflowBoundary", DefaultWorkflowBoundary);
        var prompt = BuildBoundaryPrompt(consentEnabled, summary, boundary);

        return new ProfileResponse
        {
            ApiVersion = "v1",
            UserId = userId.ToString(),
            DisplayName = user?.Username ?? user?.Email ?? "ORSA User",
            Country = "",
            Region = "",
            City = "",
            PersonaJson = data.ToJsonString(),
            PersonaUpdatedAt = ReadString(data, "personaUpdatedAt", ""),
            PersonaSummary = summary,
            WorkflowBoundary = boundary,
            ConsentStatus = consentEnabled ? "enabled" : "disabled",
            BoundaryPrompt = prompt
        };
    }

    private static DateTime ParseIsoOrUtcNow(string value)
    {
        return DateTime.TryParse(value, CultureInfo.InvariantCulture, DateTimeStyles.AdjustToUniversal, out var parsed)
            ? parsed.ToUniversalTime()
            : DateTime.UtcNow;
    }

    private static Guid ParseUserId(string userId)
    {
        if (Guid.TryParse(userId, out var parsed))
        {
            return parsed;
        }

        throw new RpcException(new Status(StatusCode.InvalidArgument, "user_id must be a UUID matching the supplied Postgres schema."));
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
