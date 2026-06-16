using System.Threading.RateLimiting;
using Microsoft.AspNetCore.RateLimiting;
using Microsoft.AspNetCore.Server.Kestrel.Core;
using Microsoft.EntityFrameworkCore;
using Npgsql;
using Orsa.SupabaseEngine.Data;
using Orsa.SupabaseEngine.Services;

// Load .env file — mirrors the Go gateway's config.go loadDotEnv().
// Walks up from cwd until a .env file is found; never overwrites existing
// process env vars so explicit system env always wins.
LoadDotEnv();

var builder = WebApplication.CreateBuilder(args);

// Bind gRPC and health on explicit ports/protocols. gRPC requires HTTP/2; over
// cleartext (no TLS) Kestrel will not negotiate h2c via ALPN, so the protocol is
// pinned per endpoint here. Ports are configurable to avoid clashing with other
// software already listening on the host (e.g. a local LicenseService on 50052).
var grpcPort = int.TryParse(builder.Configuration["GRPC_PORT"], out var parsedGrpcPort) ? parsedGrpcPort : 50053;
var healthPort = int.TryParse(builder.Configuration["HEALTH_PORT"], out var parsedHealthPort) ? parsedHealthPort : 8085;
builder.WebHost.ConfigureKestrel(options =>
{
    options.ListenAnyIP(grpcPort, listen => listen.Protocols = HttpProtocols.Http2);
    options.ListenAnyIP(healthPort, listen => listen.Protocols = HttpProtocols.Http1);
});

builder.Services.AddGrpc();
builder.Services.AddHttpClient();          // IHttpClientFactory for Google OAuth + Resend
builder.Services.AddScoped<EmailService>();  // Resend verification email
builder.Services.AddScoped<AuthService>(); // auth register/login/google handlers
builder.Services.AddScoped<UserDataService>();

// Rate-limit the auth endpoints per client IP to blunt password brute-forcing
// and registration/enumeration abuse.
builder.Services.AddRateLimiter(options =>
{
    options.RejectionStatusCode = StatusCodes.Status429TooManyRequests;
    options.AddPolicy("auth", httpContext =>
        RateLimitPartition.GetFixedWindowLimiter(
            partitionKey: httpContext.Connection.RemoteIpAddress?.ToString() ?? "unknown",
            factory: _ => new FixedWindowRateLimiterOptions
            {
                PermitLimit = 10,
                Window = TimeSpan.FromMinutes(1),
                QueueLimit = 0
            }));
});

if (SessionTokens.ResolveSecret(builder.Configuration) == SessionTokens.DevSecret)
{
    Console.WriteLine("[warn] ORSA_SESSION_SECRET not set; using the insecure dev session secret. Set it before deploying.");
}
builder.Services.AddDbContext<OrsaDbContext>(options =>
{
    var connectionString = NormalizePostgresConnectionString(builder.Configuration["SUPABASE_DB_CONNECTION_STRING"]);
    if (!string.IsNullOrWhiteSpace(connectionString))
    {
        // Supabase poolers can drop idle connections; retry transient failures so a
        // stale pooled connection does not surface as a 500 to the orchestrator.
        options.UseNpgsql(connectionString, npgsql => npgsql.EnableRetryOnFailure(
            maxRetryCount: 3,
            maxRetryDelay: TimeSpan.FromSeconds(2),
            errorCodesToAdd: null));
    }
    else
    {
        options.UseInMemoryDatabase("orsa-supabase-engine");
    }
});

var app = builder.Build();

app.UseRateLimiter();

app.MapGrpcService<UserGrpcService>();
app.MapGet("/healthz", () => Results.Ok(new { status = "ok", service = "csharp-supabase" }));

// ── Auth REST endpoints (consumed by Angular via proxy → port 8085) ──────────
// The whole group is rate-limited per IP (see the "auth" policy above).
var auth = app.MapGroup("/auth").RequireRateLimiting("auth");
auth.MapPost("/register",
    async (RegisterRequest req, AuthService svc, CancellationToken ct) =>
        await svc.RegisterAsync(req, ct));
auth.MapPost("/login",
    async (LoginRequest req, AuthService svc, CancellationToken ct) =>
        await svc.LoginAsync(req, ct));
auth.MapGet("/google",
    (HttpContext ctx, IConfiguration cfg) =>
        AuthService.GoogleRedirect(ctx, cfg));
auth.MapPost("/google/exchange",
    async (GoogleExchangeRequest req, HttpContext ctx, AuthService svc, IConfiguration cfg, CancellationToken ct) =>
        await svc.GoogleExchangeAsync(req, ctx, cfg, ct));
auth.MapPost("/verify-email",
    async (VerifyEmailRequest req, AuthService svc, CancellationToken ct) =>
        await svc.VerifyEmailAsync(req, ct));
auth.MapPost("/resend-verification",
    async (ResendVerificationRequest req, AuthService svc, CancellationToken ct) =>
        await svc.ResendVerificationAsync(req, ct));

using (var scope = app.Services.CreateScope())
{
    var db = scope.ServiceProvider.GetRequiredService<OrsaDbContext>();
    await db.Database.EnsureCreatedAsync();

    // For PostgreSQL: ensure every table and the auth columns exist. EnsureCreated
    // above is a no-op once any table (e.g. an externally-created `users`) is
    // present, so it never creates the supporting tables on its own — which left
    // user_settings / legal_acceptances / persona_audit_records missing and made
    // registration and settings/profile writes fail with "relation does not exist".
    // All statements below are idempotent and safe to run on every startup.
    if (db.Database.ProviderName?.Contains("Npgsql", StringComparison.OrdinalIgnoreCase) == true)
    {
        await db.Database.ExecuteSqlRawAsync("""
            CREATE TABLE IF NOT EXISTS users (
                id uuid PRIMARY KEY,
                email text,
                username text,
                password_hash text,
                google_sub text,
                auth_provider text NOT NULL DEFAULT 'email',
                created_at timestamptz,
                last_login timestamptz
            );
            ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash text;
            ALTER TABLE users ADD COLUMN IF NOT EXISTS pending_password_hash text;
            ALTER TABLE users ADD COLUMN IF NOT EXISTS google_sub text;
            ALTER TABLE users ADD COLUMN IF NOT EXISTS auth_provider text NOT NULL DEFAULT 'email';
            ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified boolean NOT NULL DEFAULT false;

            -- Enforce one account per email (case-insensitive), ignoring the blank
            -- placeholder used when a settings write precedes account creation.
            -- Wrapped so that pre-existing duplicate emails cannot crash startup;
            -- if creation fails the service still boots and logs a notice.
            DO $$
            BEGIN
                CREATE UNIQUE INDEX IF NOT EXISTS ux_users_email_lower
                    ON users (lower(email))
                    WHERE email IS NOT NULL AND email <> '';
            EXCEPTION WHEN others THEN
                RAISE NOTICE 'Skipped ux_users_email_lower (likely duplicate emails present): %', SQLERRM;
            END $$;

            CREATE TABLE IF NOT EXISTS email_verifications (
                id uuid PRIMARY KEY,
                user_id uuid,
                token_hash text,
                expires_at timestamptz,
                consumed_at timestamptz,
                created_at timestamptz
            );
            CREATE INDEX IF NOT EXISTS ix_email_verifications_token ON email_verifications (token_hash);
            CREATE INDEX IF NOT EXISTS ix_email_verifications_user ON email_verifications (user_id);

            CREATE TABLE IF NOT EXISTS attachment_usage (
                user_id uuid NOT NULL,
                usage_date date NOT NULL,
                count integer NOT NULL DEFAULT 0,
                PRIMARY KEY (user_id, usage_date)
            );

            CREATE TABLE IF NOT EXISTS user_settings (
                user_id uuid PRIMARY KEY,
                theme_preference text,
                health_profile_data jsonb,
                updated_at timestamptz
            );

            CREATE TABLE IF NOT EXISTS legal_acceptances (
                id uuid PRIMARY KEY,
                user_id uuid,
                terms_version text,
                privacy_version text,
                consent_version text,
                accepted_at timestamptz
            );
            CREATE INDEX IF NOT EXISTS ix_legal_acceptances_user_accepted
                ON legal_acceptances (user_id, accepted_at);

            CREATE TABLE IF NOT EXISTS persona_audit_records (
                id uuid PRIMARY KEY,
                user_id uuid,
                source_thread_ids jsonb,
                prompt_hash varchar(128),
                model_id varchar(128),
                status varchar(32),
                extracted_json jsonb,
                error text,
                run_at timestamptz
            );
            CREATE INDEX IF NOT EXISTS ix_persona_audit_user_run
                ON persona_audit_records (user_id, run_at);
            """);
    }
}

app.Run();

static string? NormalizePostgresConnectionString(string? configuredValue)
{
    if (string.IsNullOrWhiteSpace(configuredValue))
    {
        return configuredValue;
    }

    var value = configuredValue.Trim();
    if (!value.StartsWith("postgresql://", StringComparison.OrdinalIgnoreCase)
        && !value.StartsWith("postgres://", StringComparison.OrdinalIgnoreCase))
    {
        return value;
    }

    var schemeSeparator = value.IndexOf("://", StringComparison.Ordinal);
    var remainder = value[(schemeSeparator + 3)..];
    var queryStart = remainder.IndexOf('?', StringComparison.Ordinal);
    if (queryStart >= 0)
    {
        remainder = remainder[..queryStart];
    }

    var userInfoSeparator = remainder.LastIndexOf('@');
    if (userInfoSeparator <= 0)
    {
        throw new InvalidOperationException("SUPABASE_DB_CONNECTION_STRING must include PostgreSQL URI credentials.");
    }

    var userInfo = remainder[..userInfoSeparator];
    var hostAndDatabase = remainder[(userInfoSeparator + 1)..];

    var credentialSeparator = userInfo.IndexOf(':', StringComparison.Ordinal);
    var username = credentialSeparator >= 0 ? userInfo[..credentialSeparator] : userInfo;
    var password = credentialSeparator >= 0 ? userInfo[(credentialSeparator + 1)..] : string.Empty;

    var databaseSeparator = hostAndDatabase.IndexOf('/', StringComparison.Ordinal);
    var hostAndPort = databaseSeparator >= 0 ? hostAndDatabase[..databaseSeparator] : hostAndDatabase;
    var database = databaseSeparator >= 0 ? hostAndDatabase[(databaseSeparator + 1)..] : "postgres";

    var host = hostAndPort;
    var port = 5432;
    var portSeparator = hostAndPort.LastIndexOf(':');
    if (portSeparator > 0 && int.TryParse(hostAndPort[(portSeparator + 1)..], out var parsedPort))
    {
        host = hostAndPort[..portSeparator];
        port = parsedPort;
    }

    return new NpgsqlConnectionStringBuilder
    {
        Host = Uri.UnescapeDataString(host),
        Port = port,
        Database = Uri.UnescapeDataString(database),
        Username = Uri.UnescapeDataString(username),
        Password = Uri.UnescapeDataString(password),
        SslMode = SslMode.Require,
        Pooling = true
    }.ConnectionString;
}

/// <summary>
/// Walks up from the current working directory until a .env file is found, then
/// loads its key=value pairs into the process environment.  Existing OS/process
/// environment variables are never overwritten — explicit env always wins.
/// Mirrors the Go gateway's <c>loadDotEnv()</c> in internal/config/config.go.
/// </summary>
static void LoadDotEnv()
{
    var dir = Directory.GetCurrentDirectory();
    while (true)
    {
        var candidate = Path.Combine(dir, ".env");
        if (File.Exists(candidate))
        {
            foreach (var line in File.ReadLines(candidate))
            {
                var trimmed = line.Trim();
                if (string.IsNullOrEmpty(trimmed) || trimmed.StartsWith('#')) continue;

                var eq = trimmed.IndexOf('=');
                if (eq <= 0) continue;

                var key = trimmed[..eq].Trim();
                var value = trimmed[(eq + 1)..].Trim().Trim('"', '\'');

                // Only set if not already provided by the OS/process environment.
                if (string.IsNullOrEmpty(Environment.GetEnvironmentVariable(key)))
                    Environment.SetEnvironmentVariable(key, value);
            }
            return;
        }

        var parent = Path.GetDirectoryName(dir);
        if (parent is null || parent == dir) return; // reached filesystem root
        dir = parent;
    }
}
