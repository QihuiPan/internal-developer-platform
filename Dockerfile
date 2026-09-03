FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/platform-api ./cmd/platform-api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/platform-api /platform-api
USER nonroot:nonroot
EXPOSE 8080
ENV PLATFORM_ADDRESS=:8080
ENV PLATFORM_DATA_PATH=/var/lib/platform/state.json
ENV GENERATED_SERVICES_DIR=/var/lib/platform/generated
ENTRYPOINT ["/platform-api"]
