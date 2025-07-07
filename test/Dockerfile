###  Builder ###
FROM golang:1.24  as builder

ENV PKG=/workspace
WORKDIR ${PKG}

COPY go.mod go.mod
COPY go.sum go.sum
# remove this line once the replace command is removed from go.mod
COPY pkg/apis/go.mod pkg/apis/go.mod
#
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# install go-junit-report binary that will then be copied to the runner image
RUN GOBIN=/workspace/bin go install github.com/jstemmer/go-junit-report/v2@latest

# Copy the go source
COPY pkg/ pkg/
COPY test/ test/

# compile test into e2e.test binary
RUN go test -c -tags netgo,osusergo ./test/e2e/

###  Runner  ###

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
WORKDIR /workspace

COPY --from=builder /workspace/e2e.test .
COPY --from=builder /workspace/test/osd-e2e.sh .
COPY --from=builder /workspace/bin/go-junit-report .

RUN mkdir -p /test-run-results
ENTRYPOINT [ "/workspace/osd-e2e.sh" ]
