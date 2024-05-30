# ################
# 1st Stage: Use openjdk 8 to verify signature w/ jarsigner
# https://developers.redhat.com/articles/2022/09/16/updating-docker-hubs-openjdk-image#openjdk_and_java_se_updates
# ################
FROM eclipse-temurin:8-jdk-alpine AS download_verification

RUN apk -q update && \
    apk add bash && \
    apk add -q wget && \
    rm -rf /tmp/*

ARG MID_INSTALLATION_URL=https://install.service-now.com/glide/distribution/builds/package/app-signed/mid/2024/03/06/mid.vancouver-07-06-2023__patch7-hotfix1-03-05-2024_03-06-2024_1225.linux.x86-64.zip
ARG MID_INSTALLATION_FILE
ARG MID_SIGNATURE_VERIFICATION="TRUE"

WORKDIR /opt/snc_mid_server/

COPY asset/* /opt/snc_mid_server/

# download.sh and validate_signature.sh
RUN chmod 6750 /opt/snc_mid_server/*.sh

RUN echo "Check MID installer URL: ${MID_INSTALLATION_URL} or Local installer: ${MID_INSTALLATION_FILE}"

# Download the installation ZIP file or using the local one
RUN if [ -z "$MID_INSTALLATION_FILE" ] ; \
    then /opt/snc_mid_server/download.sh $MID_INSTALLATION_URL ; \
    else echo "Use local file: $MID_INSTALLATION_FILE" && ls -alF /opt/snc_mid_server/ && mv /opt/snc_mid_server/$MID_INSTALLATION_FILE /tmp/mid.zip ; fi

# Verify mid.zip signature
RUN if [ "$MID_SIGNATURE_VERIFICATION" = "TRUE" ] || [ "$MID_SIGNATURE_VERIFICATION" = "true" ] ; \
    then echo "Verify the signature of the installation file" && /opt/snc_mid_server/validate_signature.sh /tmp/mid.zip; \
    else echo "Skip signature validation of the installation file "; fi

RUN unzip -d /opt/snc_mid_server/ /tmp/mid.zip && rm -f /tmp/mid.zip

# ################
# Final Stage (using the downloaded ZIP file from previous stage)
# ################
FROM almalinux:9.1

RUN dnf install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm

RUN dnf update -y && \
    dnf install -y  --allowerasing bind-utils \
                    xmlstarlet \
                    curl \
                    procps \
                    net-tools \
                    iputils &&\
    dnf clean packages -y && \
    rm -rf /tmp/*

# ##########################
# Build argument definition
# ##########################


ARG MID_USERNAME=mid

ARG GROUP_ID=1001

ARG USER_ID=1001


# ############################
# Runtime Env Var Definition
# ############################

# Ensure UTF-8 Encoding
ENV LANG en_US.UTF-8

# Mandatory Env Var
ENV MID_INSTANCE_URL "" \
    MID_INSTANCE_USERNAME "" \
    MID_INSTANCE_PASSWORD "" \
    MID_SERVER_NAME "" \
# Optional Env Var
    MID_PROXY_HOST "" \
    MID_PROXY_PORT "" \
    MID_PROXY_USERNAME "" \
    MID_PROXY_PASSWORD "" \
    MID_SECRETS_FILE "" \
    MID_MUTUAL_AUTH_PEM_FILE "" \
    MID_SSL_BOOTSTRAP_CERT_REVOCATION_CHECK "" \
    MID_SSL_USE_INSTANCE_SECURITY_POLICY ""


RUN if [[ -z "${GROUP_ID}" ]]; then GROUP_ID=1001; fi && \
		if [[ -z "${USER_ID}" ]]; then USER_ID=1001; fi && \
        echo "Add GROUP id: ${GROUP_ID}, USER id: ${USER_ID} for username: ${MID_USERNAME}"


RUN groupadd -g $GROUP_ID $MID_USERNAME && \
        useradd -c "MID container user" -r -m -u $USER_ID -g $MID_USERNAME $MID_USERNAME

# only copy needed scripts and .container
COPY asset/init asset/.container asset/check_health.sh asset/post_start.sh asset/pre_stop.sh /opt/snc_mid_server/

# 6:setuid + setgid, 750: a:rwx, g:rx, o:
RUN chmod 6750 /opt/snc_mid_server/* && chown -R $MID_USERNAME:$MID_USERNAME /opt/snc_mid_server/

# Copy agent/ from download_verification
COPY --chown=$MID_USERNAME:$MID_USERNAME  --from=download_verification /opt/snc_mid_server/agent/ /opt/snc_mid_server/agent/

# Check if the wrapper PID file exists and a HeartBeat is processed in the last 30 minutes
HEALTHCHECK --interval=5m --start-period=3m --retries=3 --timeout=15s \
    CMD bash check_health.sh || exit 1

WORKDIR /opt/snc_mid_server/

USER $MID_USERNAME

ENTRYPOINT ["/opt/snc_mid_server/init", "start"]
