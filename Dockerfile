FROM gcr.io/distroless/static

COPY jvsctl /bin/jvsctl

# Normally we would set this to run as "nobody". But goreleaser builds the
# binary locally and sometimes it will mess up the permission and cause "exec
# user process caused: permission denied".
#
# USER nobody

# Run the CLI
ENTRYPOINT ["/bin/jvsctl"]
