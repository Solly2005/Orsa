FROM golang:1.26-bookworm AS go-build
WORKDIR /src
RUN apt-get update \
  && apt-get install -y --no-install-recommends build-essential \
  && rm -rf /var/lib/apt/lists/*
COPY services/go-ai-mongo/go.mod services/go-ai-mongo/go.sum* ./
RUN go mod download
COPY services/go-ai-mongo ./
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /out/orsa-ai-mongo ./cmd/orsa-ai-mongo

FROM mcr.microsoft.com/dotnet/sdk:9.0 AS csharp-build
WORKDIR /src
COPY proto ./proto
COPY services/csharp-supabase ./services/csharp-supabase
WORKDIR /src/services/csharp-supabase
RUN dotnet restore
RUN dotnet publish -c Release -o /out --no-restore

FROM debian:bookworm-slim AS onnxruntime
ARG ONNXRUNTIME_VERSION=1.22.0
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates curl tar \
  && rm -rf /var/lib/apt/lists/*
RUN mkdir -p /onnxruntime/lib \
  && curl -fsSL "https://github.com/microsoft/onnxruntime/releases/download/v${ONNXRUNTIME_VERSION}/onnxruntime-linux-x64-${ONNXRUNTIME_VERSION}.tgz" \
  | tar -xz -C /tmp \
  && cp "/tmp/onnxruntime-linux-x64-${ONNXRUNTIME_VERSION}/lib/libonnxruntime.so.${ONNXRUNTIME_VERSION}" /onnxruntime/lib/

FROM mcr.microsoft.com/dotnet/aspnet:9.0 AS runtime
ARG ONNXRUNTIME_VERSION=1.22.0
RUN apt-get update \
  && apt-get install -y --no-install-recommends bash ca-certificates nginx netcat-openbsd curl libgomp1 libstdc++6 \
  && rm -rf /var/lib/apt/lists/* \
  && useradd -m -u 1000 user

ENV HOME=/home/user \
  APP_HOME=/home/user/app \
  HTTP_PORT=3000 \
  GRPC_PORT=50053 \
  HEALTH_PORT=8085 \
  CSHARP_SUPABASE_GRPC_URL=127.0.0.1:50053 \
  BERT_BASE_MODEL_ID=emilyalsentzer/Bio_ClinicalBERT \
  BERT_ESI_MODEL_DIR=/home/user/app/models/bert_esi_onnx \
  BERT_ESI_ONNX_DIR=/home/user/app/models/bert_esi_onnx \
  ONNXRUNTIME_SHARED_LIBRARY_PATH=/usr/local/lib/libonnxruntime.so.${ONNXRUNTIME_VERSION} \
  ONNX_RUNTIME_LIB=/usr/local/lib/libonnxruntime.so.${ONNXRUNTIME_VERSION} \
  LD_LIBRARY_PATH=/usr/local/lib \
  GITHUB_MODELS_BASE=https://models.inference.ai.azure.com \
  VISION_MODEL_ID=meta/Llama-3.2-90B-Vision-Instruct

WORKDIR /home/user/app
COPY --from=go-build --chown=user:user /out/orsa-ai-mongo /usr/local/bin/orsa-ai-mongo
COPY --from=csharp-build --chown=user:user /out ./csharp
COPY --from=onnxruntime /onnxruntime/lib/ /usr/local/lib/
COPY --chown=user:user models/bert_esi_onnx ./models/bert_esi_onnx
COPY --chown=user:user deploy/huggingface/backend/nginx.conf ./deploy/huggingface/backend/nginx.conf
COPY --chown=user:user deploy/huggingface/backend/start.sh ./deploy/huggingface/backend/start.sh
RUN chmod +x ./deploy/huggingface/backend/start.sh

USER user
EXPOSE 7860
HEALTHCHECK CMD curl -fsS http://127.0.0.1:7860/healthz || exit 1
CMD ["./deploy/huggingface/backend/start.sh"]
