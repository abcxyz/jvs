# Use distroless for ca certs.
FROM gcr.io/distroless/static AS distroless

# Use a scratch image to host our binary.
FROM scratch

ARG APP

COPY --from=distroless /etc/passwd /etc/passwd
COPY --from=distroless /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY $APP /server

# Normally we would set this to run as "nobody".
# But goreleaser builds the binary locally and sometimes it will mess up the permission
# and cause "exec user process caused: permission denied".
#
# USER nobody

# Run the web service on container startup.
ENV PORT 8080
ENTRYPOINT ["/server"]
