FROM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

COPY hyperfleet-db/go.mod hyperfleet-db/go.sum ./hyperfleet-db/
COPY hyperfleet-operator/api/go.mod hyperfleet-operator/api/go.sum ./hyperfleet-operator/api/
COPY hyperfleet-operator/go.mod hyperfleet-operator/go.sum ./hyperfleet-operator/
RUN cd hyperfleet-operator && go mod download

COPY hyperfleet-db/ ./hyperfleet-db/
COPY hyperfleet-operator/ ./hyperfleet-operator/

RUN cd hyperfleet-operator && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -a -o /workspace/manager ./cmd/manager
RUN cd hyperfleet-operator && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -a -o /workspace/compactor ./cmd/compactor

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=builder /workspace/compactor .
USER 65532:65532

ENTRYPOINT ["/manager"]
