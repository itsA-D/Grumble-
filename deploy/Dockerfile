# syntax=docker/dockerfile:1

# --- Build stage: generate code/assets and compile a static binary -----------
FROM golang:1.26.9-bookworm AS build
WORKDIR /src

# Cache module downloads before copying the rest of the source.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Tailwind (pinned, checksum-verified) and templ must run before the build
# because the binary embeds web/static and the generated _templ.go files.
RUN go run ./scripts/tw -i web/src/app.css -o web/static/app.css --minify \
    && go tool templ generate \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app

# --- Runtime stage: distroless, non-root, static -----------------------------
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

COPY --from=build /out/app /app

USER nonroot:nonroot
EXPOSE 3000
ENTRYPOINT ["/app"]
CMD ["serve"]
