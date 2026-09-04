# Build the manager binary
FROM golang:1.26 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
COPY pkg/apis/ pkg/apis/
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/operator/main.go cmd/operator/main.go
COPY pkg/ pkg/
COPY must-gather/ must-gather/

# Build the operator and the must-gather binary
RUN GOOS=linux GOARCH=amd64 go build -a -tags netgo,osusergo -o manager cmd/operator/main.go
RUN GOOS=linux GOARCH=amd64 go build -a -tags netgo,osusergo -o gather ./must-gather/cmd

FROM registry.access.redhat.com/ubi8/ubi-micro
WORKDIR /

COPY --from=builder /workspace/manager /usr/bin/manager
# The must-gather entrypoint is a self-contained Go binary; no oc dependency.
COPY --from=builder /workspace/gather /usr/bin/gather
USER 65532:65532

ENTRYPOINT ["/usr/bin/manager"]
