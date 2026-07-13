# Build stage
FROM registry.access.redhat.com/ubi9/go-toolset:1.26 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -buildvcs=false \
    -ldflags="-w -s" \
    -o rosa-regional-platform-api \
    ./cmd/rosa-regional-platform-api

# Runtime stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

ARG VERSION=0.0.1
ARG RELEASE=1

LABEL name="rosa-hyperfleet-api" \
      vendor="Red Hat, Inc." \
      version="${VERSION}" \
      release="${RELEASE}" \
      summary="ROSA Hyperfleet platform API" \
      description="ROSA Hyperfleet platform API service" \
      io.k8s.display-name="rosa-hyperfleet-api" \
      io.k8s.description="ROSA Hyperfleet platform API service" \
      com.redhat.component="rosa-hyperfleet-api-container" \
      distribution-scope="public" \
      url="https://github.com/openshift-online/rosa-hyperfleet-api"

WORKDIR /app

COPY --from=builder /app/rosa-regional-platform-api /app/rosa-regional-platform-api

EXPOSE 8000 8081 9090

USER 1001

ENTRYPOINT ["/app/rosa-regional-platform-api"]
CMD ["serve"]
