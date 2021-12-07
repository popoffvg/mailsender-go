FROM registry.gitlab.com/gitlab-org/gitlab-build-images:golangci-lint-alpine as builder
WORKDIR /build

COPY go.* /build/
RUN go mod download
COPY . /build/

RUN CGO_ENABLED=0 GOOS=linux go build -a -o app ./cmd

# generate clean, final image for end users
FROM quay.io/jitesoft/alpine:3.11.3
WORKDIR /

COPY --from=builder /build/app .

# executable
ENTRYPOINT ["./app"]