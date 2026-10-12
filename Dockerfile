# --- frontend ---
FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# --- backend (frontend wordt via go:embed in de binary gebakken) ---
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /pvdiary ./cmd/pvdiary
RUN mkdir /data

# --- runtime ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /pvdiary /pvdiary
COPY --from=build --chown=nonroot:nonroot /data /data
ENV PVDIARY_ADDR=0.0.0.0:8080 \
    PVDIARY_DB=/data/pvdiary.db
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/pvdiary"]
CMD ["serve"]
