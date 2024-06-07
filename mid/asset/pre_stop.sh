#!/bin/bash

MID_CONTAINER_DIR="/opt/snc_mid_server/mid_container"
LOG_FILE="/opt/snc_mid_server/mid-container.log"

logInfo () {
  msg="$(date '+%Y-%m-%dT%T.%3N') ${1}"
  echo "$msg" | tee -a ${LOG_FILE}
}

# Copy the config, wrapper config and other metadata files to the persistent volume
if [[ -d $MID_CONTAINER_DIR ]]
then
  LOG_FILE="${MID_CONTAINER_DIR}/mid-container.log";
  logInfo "Current user id: `id`"
  logInfo "Backup the config and other metadata files to the persistent volume"
  \cp -f /opt/snc_mid_server/agent/config.xml \
     /opt/snc_mid_server/agent/conf/wrapper-override.conf \
     /opt/snc_mid_server/agent/.initialized \
     /opt/snc_mid_server/agent/.env_hash \
     /opt/snc_mid_server/.container \
     /opt/snc_mid_server/agent/properties/glide.properties \
     ${MID_CONTAINER_DIR}/
else
  logInfo "The directory $MID_CONTAINER_DIR does not exist!"
fi

# Create the drain marker file
DRAIN_MARKER_FILE="/opt/snc_mid_server/.drain_before_termination"
if [[ ! -f "$DRAIN_MARKER_FILE" ]]; then
  logInfo "Create the drain marker file: ${DRAIN_MARKER_FILE}"
  touch $DRAIN_MARKER_FILE
fi

# Tell the wrapper to stop the MID server. Before stop, the MID server will drain if it sees
# the drain marker file and if mid.drain.run_before_container_termination = true
logInfo "Stop the MID server"
/opt/snc_mid_server/agent/bin/mid.sh stop

# Remove the drain marker file
logInfo "Remove the drain marker file: ${DRAIN_MARKER_FILE}"
rm -f $DRAIN_MARKER_FILE
if [[ -f $DRAIN_MARKER_FILE ]]
then
  logInfo "Failed to delete ${DRAIN_MARKER_FILE}"
fi
