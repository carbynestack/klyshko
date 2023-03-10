# Dockerfile scaffolded by the Operator SDK.

# Build the manager binary
FROM golang:1.16 as builder

ARG RELEASE_PAGE="https://github.com/carbynestack/klyshko/releases"

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY castor/ castor/
COPY controllers/ controllers/
COPY logging/ logging/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

# Use compliant base image from carbynestack/base-images
FROM ghcr.io/carbynestack/ubuntu:20.04-20210827-nonroot

# Copy resources
COPY --from=builder /workspace/manager /
COPY 3RD-PARTY-LICENSES /3RD-PARTY-LICENSES

USER root:root
RUN chmod -R ugo+rXw /3RD-PARTY-LICENSES/disclosure.md
RUN printf "\n## Klyshko Operator\n\
General information about third-party software components and their licenses, \
which are distributed with Klyshko Operator, can be found in the \
[SBOM](./sbom.json). Further details are available in the subfolder for the \
respective component or can be downloaded from the \
[Klyshko Release Page](%s).\n" "${RELEASE_PAGE}"\
  >> /3RD-PARTY-LICENSES/disclosure.md

USER cs:cs

ENTRYPOINT ["/manager"]
