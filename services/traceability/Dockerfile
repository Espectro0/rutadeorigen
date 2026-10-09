ARG GO_VERSION=1.27

FROM golang:${GO_VERSION} AS build
WORKDIR /src

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/traceability ./services/traceability/cmd/traceability

FROM gcr.io/distroless/static:nonroot

LABEL org.opencontainers.image.source="https://github.com/Espectro0/rutadeorigen"
LABEL org.opencontainers.image.title="rutadeorigen-traceability"

COPY --from=build /out/traceability /app/traceability

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/app/traceability"]
