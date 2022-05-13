FROM golang:1.17-alpine as builder

WORKDIR /workspace
COPY go.* .
RUN go mod download

COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -o manager main.go

#--------------------------------------------------------------------------------------------------

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]