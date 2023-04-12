# Base image that supports bash
FROM cgr.dev/chainguard/bash:latest

COPY jvsctl /jvsctl

COPY ./prober/prober.sh /prober.sh

# Run the bash script on container startup.
ENTRYPOINT ["./prober.sh"]
