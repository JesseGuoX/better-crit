# Reference Dockerfile for running crit-plus alongside an AI coding agent
# in a sandboxed container. See README.md in this directory for the full
# explanation. The image installs:
#   - crit-plus (built from this Crit Plus checkout; based on Crit by Tomasz Tomczyk)
#   - claude code (npm)
#   - socat (bridges crit-plus's loopback-bound server to 0.0.0.0 inside the
#     container so docker -p forwarding works — crit-plus itself stays bound
#     to 127.0.0.1)

FROM golang:1.26-bookworm AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /out/crit-plus ./cmd/crit

FROM node:22-bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
        git ca-certificates socat \
    && rm -rf /var/lib/apt/lists/*
RUN npm install -g @anthropic-ai/claude-code
COPY --from=builder /out/crit-plus /usr/local/bin/crit-plus
COPY --from=builder /src/LICENSE /src/NOTICE /usr/local/share/doc/crit-plus/
COPY integrations/docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# BRIDGE_PORT: container-external port (what you map with `docker -p`)
# CRIT_PORT:   crit-plus's internal loopback port
ENV CRIT_PORT=8081 BRIDGE_PORT=8080

WORKDIR /workspace
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["bash"]
