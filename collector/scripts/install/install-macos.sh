#!/bin/sh

set -e

# fork of: https://github.com/observIQ/bindplane-agent/blob/release/v1.45.0/scripts/install/install_macos.sh

# Collector Constants
SERVICE_NAME="com.servicenow.collector"
DOWNLOAD_BASE="https://github.com/lightstep/sn-collector/releases/download"

# Script Constants
PREREQS="printf sed uname tr find grep"
TMP_DIR="${TMPDIR:-"/tmp/"}sn-collector" # Allow this to be overriden by cannonical TMPDIR env var
INSTALL_DIR="/opt/sn-collector"
SCRIPT_NAME="$0"
INDENT_WIDTH='  '
indent=""

# Colors
num_colors=$(tput colors 2>/dev/null)
if test -n "$num_colors" && test "$num_colors" -ge 8; then
  bold="$(tput bold)"
  underline="$(tput smul)"
  # standout can be bold or reversed colors dependent on terminal
  standout="$(tput smso)"
  reset="$(tput sgr0)"
  bg_black="$(tput setab 0)"
  bg_blue="$(tput setab 4)"
  bg_cyan="$(tput setab 6)"
  bg_green="$(tput setab 2)"
  bg_magenta="$(tput setab 5)"
  bg_red="$(tput setab 1)"
  bg_white="$(tput setab 7)"
  bg_yellow="$(tput setab 3)"
  fg_black="$(tput setaf 0)"
  fg_blue="$(tput setaf 4)"
  fg_cyan="$(tput setaf 6)"
  fg_green="$(tput setaf 2)"
  fg_magenta="$(tput setaf 5)"
  fg_red="$(tput setaf 1)"
  fg_white="$(tput setaf 7)"
  fg_yellow="$(tput setaf 3)"
fi

if [ -z "$reset" ]; then
  sed_ignore=''
else
  sed_ignore="/^[$reset]+$/!"
fi

# Helper Functions
printf() {
  if command -v sed >/dev/null; then
    command printf -- "$@" | sed -E "$sed_ignore s/^/$indent/g"  # Ignore sole reset characters if defined
  else
    # Ignore $* suggestion as this breaks the output
    # shellcheck disable=SC2145
    command printf -- "$indent$@"
  fi
}

increase_indent() { indent="$INDENT_WIDTH$indent" ; }
decrease_indent() { indent="${indent#*"$INDENT_WIDTH"}" ; }

# Color functions reset only when given an argument
bold() { command printf "$bold$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
underline() { command printf "$underline$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
standout() { command printf "$standout$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
# Ignore "parameters are never passed"
# shellcheck disable=SC2120
reset() { command printf "$reset$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
bg_black() { command printf "$bg_black$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
bg_blue() { command printf "$bg_blue$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
bg_cyan() { command printf "$bg_cyan$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
bg_green() { command printf "$bg_green$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
bg_magenta() { command printf "$bg_magenta$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
bg_red() { command printf "$bg_red$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
bg_white() { command printf "$bg_white$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
bg_yellow() { command printf "$bg_yellow$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_black() { command printf "$fg_black$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_blue() { command printf "$fg_blue$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_cyan() { command printf "$fg_cyan$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_green() { command printf "$fg_green$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_magenta() { command printf "$fg_magenta$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_red() { command printf "$fg_red$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_white() { command printf "$fg_white$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_yellow() { command printf "$fg_yellow$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }

# Intentionally using variables in format string
# shellcheck disable=SC2059
info() { printf "$*\\n" ; }
# Intentionally using variables in format string
# shellcheck disable=SC2059
warn() {
  increase_indent
  printf "$fg_yellow$*$reset\\n"
  decrease_indent
}
# Intentionally using variables in format string
# shellcheck disable=SC2059
error() {
  increase_indent
  printf "$fg_red$*$reset\\n"
  decrease_indent
}
# Intentionally using variables in format string
# shellcheck disable=SC2059
success() { printf "$fg_green$*$reset\\n" ; }
# Ignore 'arguments are never passed'
# shellcheck disable=SC2120
prompt() {
  if [ "$1" = 'n' ]; then
    command printf "y/$(fg_red '[n]'): "
  else
    command printf "$(fg_green '[y]')/n: "
  fi
}

separator() { printf "===================================================\\n" ; }

banner()
{
  printf "\\n"
  separator
  printf "| %s\\n" "$*" ;
  separator
}

usage()
{
  increase_indent
  USAGE=$(cat <<EOF
Usage:
  $(fg_yellow '-v, --version')
      Defines the version of the ServiceNow Collector.
      If not provided, this will default to the latest version.
      Alternatively the COLLECTOR_VERSION environment variable can be
      set to configure the agent version.
      Example: '-v 1.2.12' will download 1.2.12.

  $(fg_yellow '-r, --uninstall')
      Stops the collector services and uninstalls the collector.

  $(fg_yellow '-l, --url')
      Defines the URL that the components will be downloaded from.
      If not provided, this will default to ServiceNow Collector\'s GitHub releases.
      Example: '-l http://my.domain.org/servicenow-collector' will download from there.

  $(fg_yellow '-b, --base-url')
      Defines the base of the download URL as '{base_url}/v{version}/otelcol-servicenow_v{version}_darwin_{os_arch}.tar.gz'.
      If not provided, this will default to '$DOWNLOAD_BASE'.
      Example: '-b http://my.domain.org/servicenow-collector/binaries' will be used as the base of the download URL.
   
  $(fg_yellow '-e, --endpoint')
    Defines the Cloud Observability ingest endpoint for telemetry
      
  $(fg_yellow '-i, --ingest-token')
    Defines the token to be used when communicating with Cloud Observability

EOF
  )
  info "$USAGE"
  decrease_indent
  return 0
}

force_exit()
{
  # Exit regardless of subshell level with no "Terminated" message
  kill -PIPE $$
  # Call exit to handle special circumstances (like running script during docker container build)
  exit 1
}

error_exit()
{
  line_num=$(if [ -n "$1" ]; then command printf ":$1"; fi)
  error "ERROR ($SCRIPT_NAME$line_num): ${2:-Unknown Error}" >&2
  shift 2
  if [ -n "$0" ]; then
    increase_indent
    error "$*"
    decrease_indent
  fi
  force_exit
}

print_prereq_line()
{
  if [ -n "$2" ]; then
    command printf "\\n${indent}  - "
    command printf "[$1]: $2"
  fi
}

check_failure()
{
  if [ "$indent" != '' ]; then increase_indent; fi
  command printf "${indent}${fg_red}ERROR: %s check failed!${reset}" "$1"

  print_prereq_line "Issue" "$2"
  print_prereq_line "Resolution" "$3"
  print_prereq_line "Help Link" "$4"
  print_prereq_line "Rerun" "$5"

  command printf "\\n"
  if [ "$indent" != '' ]; then decrease_indent; fi
  force_exit
}

succeeded()
{
  increase_indent
  success "Succeeded!"
  decrease_indent
}

failed()
{
  error "Failed!"
}

# This will check all prerequisites before running an installation.
check_prereqs()
{
  banner "Checking Prerequisites"
  increase_indent
  root_check
  os_check
  dependencies_check
  success "Prerequisite check complete!"
  decrease_indent
}

# This will check if the operating system is supported.
os_check()
{
  info "Checking that the operating system is supported..."
  os_type=$(uname -s)
  case "$os_type" in
    Darwin)
      succeeded
      ;;
    *)
      failed
      error_exit "$LINENO" "The operating system $(fg_yellow "$os_type") is not supported by this script."
      ;;
  esac
}

# This checks to see if the user who is running the script has root permissions.
root_check()
{
  system_user_name=$(id -un)
  if [ "${system_user_name}" != 'root' ]
  then
    failed
    error_exit "$LINENO" "Script needs to be run as root or with sudo"
  fi
}

# This will check if the current environment has
# all required shell dependencies to run the installation.
dependencies_check()
{
  info "Checking for script dependencies..."
  FAILED_PREREQS=''
  for prerequisite in $PREREQS; do
    if command -v "$prerequisite" >/dev/null; then
      continue
    else
      if [ -z "$FAILED_PREREQS" ]; then
        FAILED_PREREQS="${fg_red}$prerequisite${reset}"
      else
        FAILED_PREREQS="$FAILED_PREREQS, ${fg_red}$prerequisite${reset}"
      fi
    fi
  done

  if [ -n "$FAILED_PREREQS" ]; then
    failed
    error_exit "$LINENO" "The following dependencies are required by this script: [$FAILED_PREREQS]"
  fi
  succeeded
}

# This will set all installation variables
# at the beginning of the script.
setup_installation()
{
  banner "Configuring Installation Variables"
  increase_indent

  set_os_arch
  set_download_urls
  set_ingest_endpoint
  set_ingest_token

  success "Configuration complete!"
  decrease_indent
}

set_os_arch()
{
  os_arch=$(uname -m)
  case "$os_arch" in 
    # arm64 strings. These are from https://stackoverflow.com/questions/45125516/possible-values-for-uname-m
    aarch64|arm64|aarch64_be|armv8b|armv8l)
      os_arch="arm64"
      ;;
    x86_64)
      os_arch="amd64"
      ;;
    *)
      # We only support arm64/amd64 architectures for macOS
      error_exit "$LINENO" "Unsupported os arch: $os_arch"
      ;;
  esac
}

# This will set the urls to use when downloading the agent and its plugins.
# These urls are constructed based on the --version flag or COLLECTOR_VERSION env variable.
# If not specified, the version defaults to whatever the latest release on github is.
set_download_urls()
{
  if [ -z "$url" ] ; then
    if [ -z "$version" ] ; then
      # shellcheck disable=SC2153
      version=$COLLECTOR_VERSION
    fi

    if [ -z "$version" ] ; then
      version=$(latest_version)
    fi

    if [ -z "$version" ] ; then
      error_exit "$LINENO" "Could not determine version to install"
    fi

    if [ -z "$base_url" ] ; then
      base_url=$DOWNLOAD_BASE
    fi

    collector_download_url="$base_url/v$version/otelcol-servicenow_${version}_darwin_${os_arch}.tar.gz"
  else
    collector_download_url="$url"
  fi
}

set_ingest_endpoint()
{
  if [ -z "$ingest_endpoint" ] ; then
    ingest_endpoint="$ENDPOINT"
  fi

  INGEST_ENDPOINT="$ingest_endpoint"
}

set_ingest_token()
{
  if [ -z "$ingest_token" ] ; then
    ingest_token=$INGEST_TOKEN
  fi

  INGEST_TOKEN="$ingest_token"
}

# latest_version gets the tag of the latest release, without the v prefix.
latest_version()
{
  curl -sSL -H"Accept: application/vnd.github.v3+json" "https://api.github.com/repos/lightstep/sn-collector/releases/latest" | \
    grep "\"tag_name\"" | \
    sed -E 's/ *"tag_name": "v([0-9]+\.[0-9]+\.[0-9+])",/\1/'
}

# set the access token, if one was specified
set_access_token()
{
    sed -i "s/YOUR_TOKEN/$INGEST_TOKEN/g" /opt/sn-collector/config.yaml
}

# This will install the package by downloading & unpacking the tarball into the install directory
install_package()
{
  banner "Installing ServiceNow Collector"
  increase_indent 

  # Remove temporary directory, if it exists
  rm -rf "$TMP_DIR"
  mkdir -p "$TMP_DIR/artifacts"

  # Download into tmp dir
  info "Downloading tarball from $collector_download_url into temporary directory..."
  curl -L "$collector_download_url" -o "$TMP_DIR/otelcol-servicenow.tar.gz" --progress-bar --fail || error_exit "$LINENO" "Failed to download package"
  succeeded

  # unpack
  info "Unpacking tarball..."
  tar -xzf "$TMP_DIR/otelcol-servicenow.tar.gz" -C "$TMP_DIR/artifacts" || error_exit "$LINENO" "Failed to unack archive $TMP_DIR/otelcol-servicenow.tar.gz"
  succeeded

  mkdir -p "$INSTALL_DIR" || error_exit "$LINENO" "Failed to create directory $INSTALL_DIR"

  info "Creating install directory structure..."
  increase_indent
  # Find all directorys in the unpackaged tar
  DIRS=$(cd "$TMP_DIR/artifacts"; find "." -type d)
  for d in $DIRS
  do
    mkdir -p "$INSTALL_DIR/$d" || error_exit "$LINENO" "Failed to create directory $INSTALL_DIR/$d"
  done
  
  # Create the storage dir; This dir is necessary for filelog based plugins
  mkdir -p "$INSTALL_DIR/storage" || error_exit "$LINENO" "Failed to create directory $INSTALL_DIR/storage"

  decrease_indent
  succeeded

  info "Copying artifacts to install directory..."
  increase_indent

  # This find command gets a list of all artifacts paths except config.yaml, logging.yaml, or opentelemetry-java-contrib-jmx-metrics.jar
  FILES=$(cd "$TMP_DIR/artifacts"; find "." -type f -not \( -name config.yaml -or -name logging.yaml -or -name opentelemetry-java-contrib-jmx-metrics.jar \))
  # Move files to install dir
  for f in $FILES
  do
    rm -rf "$INSTALL_DIR/${f:?}"
    cp "$TMP_DIR/artifacts/${f:?}" "$INSTALL_DIR/${f:?}" || error_exit "$LINENO" "Failed to copy artifact $f to install dir"
  done
  decrease_indent
  succeeded

  if [ ! -f "$INSTALL_DIR/config.yaml" ]; then
    info "Copying default config.yaml..."
    cp "$TMP_DIR/artifacts/config/otelcol-macos-hostmetrics.yaml" "$INSTALL_DIR/config.yaml" || error_exit "$LINENO" "Failed to copy default config.yaml to install dir"
    succeeded
  fi

  if [ -f "/Library/LaunchDaemons/$SERVICE_NAME.plist" ]; then
    # Existing service file, we should stop & unload first.
    info "Uninstalling existing service file..."
    launchctl unload -w "/Library/LaunchDaemons/$SERVICE_NAME.plist" > /dev/null 2>&1 || error_exit "$LINENO" "Failed to unload service file /Library/LaunchDaemons/$SERVICE_NAME.plist"
    succeeded
  fi

  # Install service file
  info "Installing service file..."
  sed "s|\\[INSTALLDIR\\]|${INSTALL_DIR}/|g" "$INSTALL_DIR/install/$SERVICE_NAME.plist" | tee "/Library/LaunchDaemons/$SERVICE_NAME.plist" > /dev/null 2>&1 || error_exit "$LINENO" "Failed to install service file"
  launchctl load -w "/Library/LaunchDaemons/$SERVICE_NAME.plist" > /dev/null 2>&1 || error_exit "$LINENO" "Failed to load service file /Library/LaunchDaemons/$SERVICE_NAME.plist"
  succeeded

  if [ -n "$INGEST_TOKEN" ]; then
    info "Setting access token..."
    set_access_token
  fi

  info "Starting service..."
  launchctl start "$SERVICE_NAME" || error_exit "$LINENO" "Failed to start service file $SERVICE_NAME"
  succeeded

  info "Removing temporary files..."
  rm -rf "$TMP_DIR" || error_exit "$LINENO" "Failed to remove temp dir: $TMP_DIR"
  succeeded

  success "ServiceNow Collector installation complete!"
  decrease_indent
}

# This will display the results of an installation
display_results()
{
    banner 'Information'
    increase_indent
    info "Collector Home:         $(fg_cyan "$INSTALL_DIR")$(reset)"
    info "Collector Config:       $(fg_cyan "$INSTALL_DIR/config.yaml")$(reset)"
    info "Start Command:      $(fg_cyan "sudo launchctl load /Library/LaunchDaemons/$SERVICE_NAME.plist")$(reset)"
    info "Stop Command:       $(fg_cyan "sudo launchctl unload /Library/LaunchDaemons/$SERVICE_NAME.plist")$(reset)"
    info "Logs Command:       $(fg_cyan "sudo tail -F $INSTALL_DIR/log/collector.log")$(reset)"
    decrease_indent

    banner 'Support'
    increase_indent
    info "For more information on configuring the agent, see the docs:"
    increase_indent
    info "$(fg_cyan "https://github.com/lightstep/sn-collector/tree/main")$(reset)"
    decrease_indent
    info "If you have any other questions please contact us at $(fg_cyan support@lightstep.com)$(reset)"
    decrease_indent

    banner "$(fg_green Installation Complete!)"
    return 0
}

uninstall()
{
  banner "Uninstalling ServiceNow Collector"
  increase_indent

  if [ ! -f "$INSTALL_DIR/otelcol-servicenow" ]; then
    # If the agent binary is not present, we assume that the agent is not installed
    # In this case, do nothing.
    info "No install detected, skipping..."
    decrease_indent
    banner "$(fg_green Uninstallation Complete!)"
    return 0
  fi

  info "Uninstalling service file..."
  launchctl unload -w "/Library/LaunchDaemons/$SERVICE_NAME.plist" || error_exit "$LINENO" "Failed to unload service file /Library/LaunchDaemons/$SERVICE_NAME.plist"
  rm -f "/Library/LaunchDaemons/$SERVICE_NAME.plist" || error_exit "$LINENO" "Failed to remove service file /Library/LaunchDaemons/$SERVICE_NAME.plist"
  succeeded

  info "Backing up config.yaml to config.bak.yaml"
  cp "$INSTALL_DIR/config.yaml" "$INSTALL_DIR/config.bak.yaml" || error_exit "$LINENO" "Failed to backup config.yaml to config.bak.yaml"
  succeeded

  # Removes the whole install directory, including configs.
  info "Removing installed artifacts..."
  # This find command gets a list of all artifacts paths except config.yaml or the root directory.
  FILES=$(cd "$INSTALL_DIR"; find "." -not \( -name config.bak.yaml -or -name "." \))
  for f in $FILES
  do
    rm -rf "${INSTALL_DIR:?}/$f" || error_exit "$LINENO" "Failed to remove artifact ${INSTALL_DIR:?}/$f"
  done

  # Copy the old config to a backup. This is similar to how RPM handles this.
  # This way, the file is recoverable if the uninstall was somehow accidental,
  # but if a new install occurs, the default config will still be used.
  succeeded

  decrease_indent
  banner "$(fg_green Uninstallation Complete!)"
}

main()
{
  # We do these checks before we process arguments, because
  # some of these options bail early, and we'd like to be sure that those commands
  # (e.g. uninstall) can run

  
  check_prereqs

  if [ $# -ge 1 ]; then
    while [ -n "$1" ]; do
      case "$1" in
        -l|--url)
          url=$2 ; shift 2 ;;
        -v|--version)
          version=$2 ; shift 2 ;;
        -r|--uninstall)
          uninstall
          exit 0
          ;;
        -h|--help)
          usage
          exit 0
          ;;
        -e|--endpoint)
          ingest_endpoint=$2 ; shift 2 ;;
        -i|--ingest-token)
          ingest_token=$2 ; shift 2 ;;
        -b|--base-url)
          base_url=$2 ; shift 2 ;;
      --)
        shift; break ;;
      *)
        error "Invalid argument: $1"
        usage
        force_exit
        ;;
      esac
    done
  fi

  setup_installation
  install_package
  display_results
}

main "$@"