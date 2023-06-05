FROM golang:1.19 as build

WORKDIR /go/src/app
COPY . .

RUN CGO_ENABLED=0 go build -o /go/bin/app

FROM gcr.io/distroless/static-debian11
COPY --from=build /go/bin/app /mc2bq
COPY migrationcenter_v1.schema.json /
ENTRYPOINT ["/mc2bq", "-schema-path", "/migrationcenter_v1.schema.json"]

