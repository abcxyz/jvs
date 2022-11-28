# Use distroless for ca certs.
FROM gcr.io/distroless/static AS distroless

# Use a scratch image to host our binary.
FROM scratch

ARG APP

COPY --from=distroless /etc/passwd /etc/passwd
COPY --from=distroless /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY $APP /server

USER nobody

# Run the web service on container startup.
ENV PORT 8080
ENTRYPOINT ["/server"]
