FROM golang:1.18-alpine as builder

WORKDIR /workspace
COPY go.* .
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -o manager cmd/main.go

#--------------------------------------------------------------------------------------------------

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
