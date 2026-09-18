# Presidio Analyzer with English + Chinese spaCy md models for Flowbot LLM PII.
# Base image: https://github.com/data-privacy-stack/presidio
#
# Pin a digest for reproducible deploys, e.g.:
#   docker build --build-arg PRESIDIO_ANALYZER_IMAGE=mcr.microsoft.com/presidio-analyzer@sha256:<digest> ...
ARG PRESIDIO_ANALYZER_IMAGE=mcr.microsoft.com/presidio-analyzer:latest
FROM ${PRESIDIO_ANALYZER_IMAGE}

USER root

COPY deployments/presidio/conf/nlp.yaml /conf/nlp.yaml
COPY deployments/presidio/conf/analyzer.yaml /conf/analyzer.yaml
COPY deployments/presidio/conf/recognizers.yaml /conf/recognizers.yaml

# Keep absolute /conf paths for ENV and mirror into the base image layout.
RUN mkdir -p /app/presidio_analyzer/conf \
    && cp /conf/nlp.yaml /app/presidio_analyzer/conf/default.yaml \
    && cp /conf/analyzer.yaml /app/presidio_analyzer/conf/default_analyzer.yaml \
    && cp /conf/recognizers.yaml /app/presidio_analyzer/conf/default_recognizers.yaml \
    && chown -R 1001:1001 /conf /app/presidio_analyzer/conf

ENV NLP_CONF_FILE=/conf/nlp.yaml \
    ANALYZER_CONF_FILE=/conf/analyzer.yaml \
    RECOGNIZER_REGISTRY_CONF_FILE=/conf/recognizers.yaml \
    PORT=3000 \
    WORKERS=1

# Base image may already ship en_core_web_lg; install the md pair declared in nlp.yaml.
# zh models pull spacy-pkuseg via the model package. Verify load so a bad install fails the build.
RUN set -eux; \
    if command -v poetry >/dev/null 2>&1; then PY="poetry run python"; else PY="python"; fi; \
    $PY -m spacy download en_core_web_md; \
    $PY -m spacy download zh_core_web_md; \
    $PY -c "import spacy; spacy.load('en_core_web_md'); spacy.load('zh_core_web_md'); print('spacy md models ok')"

USER 1001

EXPOSE 3000

# md models need a longer cold start than sm.
HEALTHCHECK --interval=30s --timeout=5s --start-period=300s --retries=5 \
  CMD curl -f http://127.0.0.1:3000/health || exit 1

LABEL org.opencontainers.image.title="flowbot-presidio-analyzer" \
      org.opencontainers.image.description="Presidio Analyzer (en+zh md) for Flowbot LLM PII anonymization" \
      org.opencontainers.image.source="https://github.com/flowline-io/flowbot"
