FROM --platform=${BUILDPLATFORM} cgr.dev/chainguard/bash:latest AS builder
ARG NAME
ARG VERSION
ARG TARGETOS
ARG TARGETARCH
COPY dist/${NAME}_${VERSION}_${TARGETOS}_${TARGETARCH} /bin/service
RUN chmod +x /bin/service
RUN echo "nobody:x:65532:65532:nobody:/nonexistent:/bin/false" > /etc/passwd.min


FROM --platform=${BUILDPLATFORM} scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd.min /etc/passwd
COPY --from=builder /bin/service /bin/service
USER nobody
ENTRYPOINT ["/bin/service"]
