FROM alpine:3.16 as certs
RUN apk --update add ca-certificates

FROM scratch

ARG USER_UID=10001
USER ${USER_UID}

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --chmod=755 otelcol-servicenow /otelcol-servicenow

COPY collector/config/otelcol-docker-hostmetrics.yaml /etc/otelcol/config.yaml

ENTRYPOINT ["/otelcol-servicenow"]
CMD ["--config", "/etc/otelcol/config.yaml"]
EXPOSE 4317 55678 55679