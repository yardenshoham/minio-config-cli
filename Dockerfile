FROM scratch
COPY minio-config-cli /
ENTRYPOINT ["/minio-config-cli"]