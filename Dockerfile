FROM golang:1.23 as builder

ENV GOOS=linux
WORKDIR /src

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o console-github-auth ./cmd/console-github-auth/

FROM cgr.dev/chainguard/static
COPY --from=builder /src/console-github-auth /app/console-github-auth
ENTRYPOINT ["/app/console-github-auth"]

