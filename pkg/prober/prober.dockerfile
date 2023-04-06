
FROM cgr.dev/chainguard/bash:latest

COPY jvsctl /jvsctl
COPY ./pkg/prober/prober.sh prober.sh

# Normally we would set this to run as "nobody".
# But goreleaser builds the binary locally and sometimes it will mess up the permission
# and cause "exec user process caused: permission denied".
#
# USER nobody

# Run the web service on container startup.

ENTRYPOINT ["bash", "prober.sh"]
