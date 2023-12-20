FROM golang as build
WORKDIR /app

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build


FROM alpine
WORKDIR /app
COPY --from=build /app/golabber /app
ENTRYPOINT [ "/app/golabber" ]


# FROM scratch as final
# COPY --from=build /app/golabber /
# ENTRYPOINT ["/golabber"]
# # # CMD /app/init.sh