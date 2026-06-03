ARG GO_VERSION=1.26
FROM golang:${GO_VERSION} AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o console-github-auth .

FROM gcr.io/distroless/base:nonroot
COPY --from=builder /src/console-github-auth /app/console-github-auth
ENTRYPOINT ["/app/console-github-auth"]

