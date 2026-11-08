# The API image for Cloud Run (issue #10): built by api.yml with buildx and the
# GHA cache, pushed to Artifact Registry tagged with the commit SHA.
# A Dockerfile over ko/buildpacks: #10 chose it, it's the same shape as
# doula-cloud, and buildx's GHA layer cache keeps `go mod download` warm.
FROM golang:1.27.3 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Static binary: distroless/static has no libc. time/tzdata is embedded by
# internal/clock, so America/New_York works without zoneinfo in the image.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /api .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /api /api
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/api"]
