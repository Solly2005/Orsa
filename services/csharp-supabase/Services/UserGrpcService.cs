using Grpc.Core;
using Orsa.Proto.User.V1;

namespace Orsa.SupabaseEngine.Services;

public sealed class UserGrpcService(UserDataService users) : UserService.UserServiceBase
{
    public override async Task<UserSettings> GetSettings(UserIdRequest request, ServerCallContext context)
    {
        var result = await users.GetSettingsAsync(ParseUserId(request.UserId), context.CancellationToken);
        return ToProto(result);
    }

    public override async Task<UserSettings> UpdateSettings(UpdateSettingsRequest request, ServerCallContext context)
    {
        var result = await users.UpdateSettingsAsync(
            ParseUserId(request.UserId),
            request.HasMemoryExtractionEnabled ? request.MemoryExtractionEnabled : null,
            request.HasRemindersEnabled ? request.RemindersEnabled : null,
            context.CancellationToken);
        return ToProto(result);
    }

    public override async Task<ProfileResponse> GetProfile(UserIdRequest request, ServerCallContext context)
    {
        var result = await users.GetProfileAsync(ParseUserId(request.UserId), context.CancellationToken);
        return ToProto(result);
    }

    public override async Task<ProfileResponse> UpdateProfile(UpdateProfileRequest request, ServerCallContext context)
    {
        var result = await users.UpdateProfileAsync(
            ParseUserId(request.UserId),
            request.HasMemoryExtractionEnabled ? request.MemoryExtractionEnabled : null,
            request.HasPersonaSummary ? request.PersonaSummary : null,
            request.HasWorkflowBoundary ? request.WorkflowBoundary : null,
            context.CancellationToken);
        return ToProto(result);
    }

    public override async Task<WriteAck> RecordLegalAcceptance(LegalAcceptanceRequest request, ServerCallContext context)
    {
        var result = await users.RecordLegalAcceptanceAsync(
            ParseUserId(request.UserId),
            request.TermsVersion,
            request.PrivacyVersion,
            request.ConsentVersion,
            request.AcceptedAtIso,
            context.CancellationToken);
        return ToProto(result);
    }

    public override async Task<WriteAck> WritePersonaAudit(PersonaAuditRequest request, ServerCallContext context)
    {
        var result = await users.WritePersonaAuditAsync(
            ParseUserId(request.UserId),
            request.SourceThreadIds.ToArray(),
            request.PromptHash,
            request.ModelId,
            request.Status,
            request.ExtractedJson,
            request.Error,
            request.RunAtIso,
            context.CancellationToken);
        return ToProto(result);
    }

    public override async Task<AttachmentUsage> GetAttachmentUsage(UserIdRequest request, ServerCallContext context)
    {
        var result = await users.GetAttachmentUsageAsync(
            ParseUserId(request.UserId),
            UserDataService.DefaultAttachmentLimit,
            context.CancellationToken);
        return ToProto(result);
    }

    public override async Task<AttachmentUsage> ConsumeAttachment(ConsumeAttachmentRequest request, ServerCallContext context)
    {
        var result = await users.ConsumeAttachmentAsync(
            ParseUserId(request.UserId),
            request.Count,
            request.Limit,
            context.CancellationToken);
        return ToProto(result);
    }

    public override async Task<WriteAck> DeleteUser(UserIdRequest request, ServerCallContext context)
    {
        var result = await users.DeleteUserAsync(ParseUserId(request.UserId), context.CancellationToken);
        return ToProto(result);
    }

    private static AttachmentUsage ToProto(AttachmentUsageView usage) => new()
    {
        ApiVersion = usage.ApiVersion,
        UserId = usage.UserId,
        UsedToday = usage.UsedToday,
        Limit = usage.Limit,
        Allowed = usage.Allowed,
        ResetAtIso = usage.ResetAtIso
    };

    private static UserSettings ToProto(UserSettingsView settings) => new()
    {
        ApiVersion = settings.ApiVersion,
        UserId = settings.UserId,
        MemoryExtractionEnabled = settings.MemoryExtractionEnabled,
        RemindersEnabled = settings.RemindersEnabled,
        AttachmentCountToday = settings.AttachmentCountToday,
        AttachmentLimit = settings.AttachmentLimit
    };

    private static ProfileResponse ToProto(ProfileView profile) => new()
    {
        ApiVersion = profile.ApiVersion,
        UserId = profile.UserId,
        DisplayName = profile.DisplayName,
        Country = profile.Country,
        Region = profile.Region,
        City = profile.City,
        PersonaJson = profile.PersonaJson,
        PersonaUpdatedAt = profile.PersonaUpdatedAt,
        PersonaSummary = profile.PersonaSummary,
        WorkflowBoundary = profile.WorkflowBoundary,
        ConsentStatus = profile.ConsentStatus,
        BoundaryPrompt = profile.BoundaryPrompt
    };

    private static WriteAck ToProto(WriteAckView ack) => new()
    {
        ApiVersion = ack.ApiVersion,
        Ok = ack.Ok,
        Id = ack.Id
    };

    private static Guid ParseUserId(string userId)
    {
        if (Guid.TryParse(userId, out var parsed))
        {
            return parsed;
        }

        throw new RpcException(new Status(StatusCode.InvalidArgument, "user_id must be a UUID matching the supplied Postgres schema."));
    }
}
