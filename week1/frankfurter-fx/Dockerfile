# ---- build stage ----
# Build on the native platform, cross-compile to the deploy target (Render/Fly shared CPU = linux/amd64).
FROM --platform=$BUILDPLATFORM golang:1.25 AS build
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -o /out/frankfurter-fx .

# ---- run stage ----
# distroless/static bundles CA certs, required for TLS to upstream / facilitator / RPC.
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/frankfurter-fx /frankfurter-fx
EXPOSE 8080
ENTRYPOINT ["/frankfurter-fx"]
