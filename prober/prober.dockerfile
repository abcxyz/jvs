# Pull the parent image so we can get the JVS
ARG PARENT
FROM --platform=${BUILDPLATFORM} ${PARENT} AS jvs

# Base image that supports bash
FROM --platform=${BUILDPLATFORM} cgr.dev/chainguard/bash:latest

COPY --from=jvs /bin/service /jvsctl
COPY ./prober/prober.sh /prober.sh

# Run the bash script on container startup.
ENTRYPOINT ["./prober.sh"]
