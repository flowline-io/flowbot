# Lightpanda CDP server for chat agent browser_* tools.
# Official image: https://hub.docker.com/r/lightpanda/browser
#
# Pin a digest for reproducible deploys, e.g.:
#   docker build --build-arg LIGHTPANDA_IMAGE=lightpanda/browser@sha256:<digest> ...
ARG LIGHTPANDA_IMAGE=lightpanda/browser:nightly
FROM ${LIGHTPANDA_IMAGE}

# Bind all interfaces inside the compose network; advertise the service DNS name
# so /json/version WebSocket URLs are reachable from sibling containers.
CMD ["lightpanda", "serve", "--host", "0.0.0.0", "--port", "9222", "--advertise-host", "lightpanda"]
