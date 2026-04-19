FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /manager ./cmd/manager
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /diagctl  ./cmd/diagctl


FROM gcr.io/distroless/static-debian12:nonroot AS manager
WORKDIR /
COPY --from=build /manager /manager
USER 65532:65532
ENTRYPOINT ["/manager"]


FROM gcr.io/distroless/static-debian12:nonroot AS diagctl
WORKDIR /
COPY --from=build /diagctl /diagctl
USER 65532:65532
ENTRYPOINT ["/diagctl"]
