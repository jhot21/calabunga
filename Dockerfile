# Stage 1: build the React frontend
FROM --platform=$BUILDPLATFORM node:lts AS web
WORKDIR /app
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: build the Go binary (embeds web/dist)
FROM --platform=$BUILDPLATFORM golang:bookworm AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /bin/calabunga ./cmd/server

# Stage 3: run unit tests against the built sources
FROM build AS test
RUN go test ./...

# Stage 4: minimal production image
FROM scratch AS final
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /bin/calabunga /bin/calabunga
EXPOSE 8080
ENTRYPOINT ["/bin/calabunga"]
