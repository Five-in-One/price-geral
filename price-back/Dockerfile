# Build: compila o binário estático da API.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Runtime: imagem pequena, só com o binário (as migrations vão embutidas nele).
FROM alpine:3.22
RUN adduser -D -H app
USER app
COPY --from=build /out/api /usr/local/bin/api
ENV PORT=8080
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --start-period=10s --retries=5   CMD wget -qO- http://127.0.0.1:8080/health >/dev/null || exit 1
ENTRYPOINT ["/usr/local/bin/api"]
