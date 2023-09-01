FROM alpine:latest

ARG VERSION=0.1
ENV VERSION=$VERSION

LABEL version=$VERSION

ENV FLOWBOT_URL=""

RUN apk add --no-cache bash

WORKDIR /opt/app

# Fetch build from Github.
COPY flowbot .
COPY deployments/docker/config.template ./flowbot.yaml
COPY deployments/docker/entrypoint.sh .

RUN chmod +x flowbot && chmod +x entrypoint.sh
RUN mkdir subscribe
RUN ls -all
RUN pwd
RUN stat flowbot
RUN stat entrypoint.sh

ENTRYPOINT ./entrypoint.sh

# HTTP, gRPC, cluster ports
EXPOSE 6060 16060 12000-12003
