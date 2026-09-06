FROM golang:1.26-alpine AS build
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
COPY web ./web
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" -o /out/platform-api ./cmd/platform-api \
    && mkdir -p /out/platform-data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/platform-api /platform-api
COPY --from=build --chown=65532:65532 /out/platform-data /var/lib/platform
USER nonroot:nonroot
EXPOSE 8080
ENV PLATFORM_ADDRESS=:8080
ENV PLATFORM_DATA_PATH=/var/lib/platform/state.json
ENV GENERATED_SERVICES_DIR=/var/lib/platform/generated
ENTRYPOINT ["/platform-api"]
