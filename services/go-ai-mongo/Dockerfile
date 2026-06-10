# Build from repo root: docker build -f services/go-ai-mongo/Dockerfile -t orsa-go-ai-mongo .

FROM golang:1.26-bookworm AS build
WORKDIR /src
RUN apt-get update \
  && apt-get install -y --no-install-recommends build-essential \
  && rm -rf /var/lib/apt/lists/*
COPY services/go-ai-mongo/go.mod services/go-ai-mongo/go.sum* ./
RUN go mod download
COPY services/go-ai-mongo ./
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /out/orsa-ai-mongo ./cmd/orsa-ai-mongo

FROM debian:bookworm-slim AS onnxruntime
ARG ONNXRUNTIME_VERSION=1.22.0
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates curl tar \
  && rm -rf /var/lib/apt/lists/*
RUN mkdir -p /onnxruntime/lib \
  && curl -fsSL "https://github.com/microsoft/onnxruntime/releases/download/v${ONNXRUNTIME_VERSION}/onnxruntime-linux-x64-${ONNXRUNTIME_VERSION}.tgz" \
  | tar -xz -C /tmp \
  && cp "/tmp/onnxruntime-linux-x64-${ONNXRUNTIME_VERSION}/lib/libonnxruntime.so.${ONNXRUNTIME_VERSION}" /onnxruntime/lib/

FROM debian:bookworm-slim AS runtime
ARG ONNXRUNTIME_VERSION=1.22.0
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates libgomp1 libstdc++6 netcat-openbsd \
  && rm -rf /var/lib/apt/lists/* \
  && groupadd --system orsa \
  && useradd --system --gid orsa --home-dir /app --shell /usr/sbin/nologin orsa
WORKDIR /app
COPY --from=build /out/orsa-ai-mongo /usr/local/bin/orsa-ai-mongo
COPY --from=onnxruntime /onnxruntime/lib/ /usr/local/lib/
COPY models ./models
ENV HTTP_PORT=3000
ENV BERT_BASE_MODEL_ID=emilyalsentzer/Bio_ClinicalBERT
ENV BERT_ESI_MODEL_DIR=/app/models/bert_esi
ENV BERT_ESI_ONNX_DIR=/app/models/bert_esi_onnx
ENV ONNXRUNTIME_SHARED_LIBRARY_PATH=/usr/local/lib/libonnxruntime.so.${ONNXRUNTIME_VERSION}
ENV ONNX_RUNTIME_LIB=/usr/local/lib/libonnxruntime.so.${ONNXRUNTIME_VERSION}
ENV GITHUB_MODELS_BASE=https://models.inference.ai.azure.com
ENV VISION_MODEL_ID=meta/Llama-3.2-90B-Vision-Instruct
USER orsa
EXPOSE 3000
HEALTHCHECK CMD nc -z 127.0.0.1 3000 || exit 1
CMD ["orsa-ai-mongo"]
