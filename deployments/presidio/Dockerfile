# Presidio Analyzer with English + Chinese spaCy models for Flowbot LLM PII.
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

ENV NLP_CONF_FILE=/conf/nlp.yaml \
    ANALYZER_CONF_FILE=/conf/analyzer.yaml \
    RECOGNIZER_REGISTRY_CONF_FILE=/conf/recognizers.yaml \
    PORT=3000

# Official image ships en_core_web_lg; install the sm pair declared in nlp.yaml.
RUN python -m spacy download en_core_web_sm \
    && python -m spacy download zh_core_web_sm

USER 1001

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=5s --start-period=120s --retries=5 \
  CMD curl -f http://127.0.0.1:3000/health || exit 1

LABEL org.opencontainers.image.title="flowbot-presidio-analyzer" \
      org.opencontainers.image.description="Presidio Analyzer (en+zh) for Flowbot LLM PII anonymization" \
      org.opencontainers.image.source="https://github.com/flowline-io/flowbot"
