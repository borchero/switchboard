FROM --platform=${BUILDPLATFORM} golang:1.22-alpine as builder

WORKDIR /workspace
COPY go.* .
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/

ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -a -o manager cmd/main.go

#--------------------------------------------------------------------------------------------------

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
