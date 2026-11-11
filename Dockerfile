# ── build stage ──────────────────────────────────────────────
FROM golang:1.24-alpine AS build
WORKDIR /src

# 依存を先に解決してレイヤーキャッシュを効かせる
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/sf-meter-api .

# ── run stage ────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/sf-meter-api /sf-meter-api
ENV PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/sf-meter-api"]
