# Build from repo root: docker build -f services/csharp-supabase/Dockerfile -t orsa-csharp-supabase .
FROM mcr.microsoft.com/dotnet/sdk:9.0 AS build
WORKDIR /src
COPY proto ./proto
COPY services/csharp-supabase ./services/csharp-supabase
WORKDIR /src/services/csharp-supabase
RUN dotnet restore
RUN dotnet publish -c Release -o /out --no-restore

FROM mcr.microsoft.com/dotnet/aspnet:9.0 AS runtime
WORKDIR /app
RUN adduser --disabled-password --gecos "" --uid 10001 orsa
COPY --from=build /out .
USER orsa
# gRPC (HTTP/2) on 50053, health/auth REST (HTTP/1) on 8085 — see Program.cs.
EXPOSE 8085 50053
HEALTHCHECK CMD wget -qO- http://127.0.0.1:8085/healthz || exit 1
ENTRYPOINT ["dotnet", "Orsa.SupabaseEngine.dll"]
