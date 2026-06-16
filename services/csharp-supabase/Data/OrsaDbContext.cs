using Microsoft.EntityFrameworkCore;
using Orsa.SupabaseEngine.Entities;

namespace Orsa.SupabaseEngine.Data;

public sealed class OrsaDbContext(DbContextOptions<OrsaDbContext> options) : DbContext(options)
{
    public DbSet<UserEntity> Users => Set<UserEntity>();
    public DbSet<UserSettingsEntity> UserSettings => Set<UserSettingsEntity>();
    public DbSet<LegalAcceptanceEntity> LegalAcceptances => Set<LegalAcceptanceEntity>();
    public DbSet<PersonaAuditEntity> PersonaAudits => Set<PersonaAuditEntity>();
    public DbSet<EmailVerificationEntity> EmailVerifications => Set<EmailVerificationEntity>();
    public DbSet<AttachmentUsageEntity> AttachmentUsages => Set<AttachmentUsageEntity>();

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        modelBuilder.Entity<UserEntity>(entity =>
        {
            entity.ToTable("users");
            entity.HasKey(x => x.Id);
            entity.Property(x => x.Id).HasColumnName("id");
            entity.Property(x => x.Email).HasColumnName("email").HasColumnType("text");
            entity.Property(x => x.Username).HasColumnName("username").HasColumnType("text");
            entity.Property(x => x.PasswordHash).HasColumnName("password_hash").HasColumnType("text").IsRequired(false);
            entity.Property(x => x.PendingPasswordHash).HasColumnName("pending_password_hash").HasColumnType("text").IsRequired(false);
            entity.Property(x => x.GoogleSub).HasColumnName("google_sub").HasColumnType("text").IsRequired(false);
            entity.Property(x => x.AuthProvider).HasColumnName("auth_provider").HasColumnType("text").HasDefaultValue("email");
            entity.Property(x => x.EmailVerified).HasColumnName("email_verified").HasColumnType("boolean").HasDefaultValue(false);
            entity.Property(x => x.CreatedAtUtc).HasColumnName("created_at").HasColumnType("timestamptz");
            entity.Property(x => x.LastLoginUtc).HasColumnName("last_login").HasColumnType("timestamptz");
        });

        modelBuilder.Entity<EmailVerificationEntity>(entity =>
        {
            entity.ToTable("email_verifications");
            entity.HasKey(x => x.Id);
            entity.HasIndex(x => x.TokenHash);
            entity.HasIndex(x => x.UserId);
            entity.Property(x => x.Id).HasColumnName("id");
            entity.Property(x => x.UserId).HasColumnName("user_id");
            entity.Property(x => x.TokenHash).HasColumnName("token_hash").HasColumnType("text");
            entity.Property(x => x.ExpiresAtUtc).HasColumnName("expires_at").HasColumnType("timestamptz");
            entity.Property(x => x.ConsumedAtUtc).HasColumnName("consumed_at").HasColumnType("timestamptz").IsRequired(false);
            entity.Property(x => x.CreatedAtUtc).HasColumnName("created_at").HasColumnType("timestamptz");
        });

        modelBuilder.Entity<AttachmentUsageEntity>(entity =>
        {
            entity.ToTable("attachment_usage");
            entity.HasKey(x => new { x.UserId, x.UsageDate });
            entity.Property(x => x.UserId).HasColumnName("user_id");
            entity.Property(x => x.UsageDate).HasColumnName("usage_date").HasColumnType("date");
            entity.Property(x => x.Count).HasColumnName("count");
        });

        modelBuilder.Entity<UserSettingsEntity>(entity =>
        {
            entity.ToTable("user_settings");
            entity.HasKey(x => x.UserId);
            entity.Property(x => x.UserId).HasColumnName("user_id");
            entity.Property(x => x.ThemePreference).HasColumnName("theme_preference").HasColumnType("text");
            entity.Property(x => x.HealthProfileData).HasColumnName("health_profile_data").HasColumnType("jsonb");
            entity.Property(x => x.UpdatedAtUtc).HasColumnName("updated_at").HasColumnType("timestamptz");
            entity.HasOne<UserEntity>()
                .WithOne()
                .HasForeignKey<UserSettingsEntity>(x => x.UserId)
                .OnDelete(DeleteBehavior.Cascade);
        });

        modelBuilder.Entity<LegalAcceptanceEntity>(entity =>
        {
            entity.ToTable("legal_acceptances");
            entity.HasKey(x => x.Id);
            entity.HasIndex(x => new { x.UserId, x.AcceptedAtUtc });
            entity.Property(x => x.Id).HasColumnName("id");
            entity.Property(x => x.UserId).HasColumnName("user_id");
            entity.Property(x => x.TermsVersion).HasColumnName("terms_version");
            entity.Property(x => x.PrivacyVersion).HasColumnName("privacy_version");
            entity.Property(x => x.ConsentVersion).HasColumnName("consent_version");
            entity.Property(x => x.AcceptedAtUtc).HasColumnName("accepted_at").HasColumnType("timestamptz");
        });

        modelBuilder.Entity<PersonaAuditEntity>(entity =>
        {
            entity.ToTable("persona_audit_records");
            entity.HasKey(x => x.Id);
            entity.HasIndex(x => new { x.UserId, x.RunAtUtc });
            entity.Property(x => x.Id).HasColumnName("id");
            entity.Property(x => x.UserId).HasColumnName("user_id");
            entity.Property(x => x.SourceThreadIdsJson).HasColumnName("source_thread_ids").HasColumnType("jsonb");
            entity.Property(x => x.PromptHash).HasColumnName("prompt_hash").HasMaxLength(128);
            entity.Property(x => x.ModelId).HasColumnName("model_id").HasMaxLength(128);
            entity.Property(x => x.Status).HasColumnName("status").HasMaxLength(32);
            entity.Property(x => x.ExtractedJson).HasColumnName("extracted_json").HasColumnType("jsonb");
            entity.Property(x => x.Error).HasColumnName("error");
            entity.Property(x => x.RunAtUtc).HasColumnName("run_at").HasColumnType("timestamptz");
        });
    }
}
