#!/bin/bash
#
# Configuration options (set by environment variables during script execution)
#   - MTG_CONFIG    - directory where mtg stores its configuration
#   - MTG_IMAGENAME - a name of the docker image to use
#   - MTG_PORT      - which port of the host system should be used
#   - MTG_CONTAINER - a name of the container to use
#
# Example:
#   export MTG_CONFIG="$HOME/mtg_config"
#   export MTG_IMAGENAME="nineseconds/mtg:latest"
#   curl -sfL --compressed https://raw.githubusercontent.com/9seconds/mtg/master/run.sh | bash

set -eu

export XDG_CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config}"
export MTG_CONFIG="${MTG_CONFIG:-$XDG_CONFIG_HOME/mtg}"

if ! [ -x "$(command -v docker)" ]; then
    echo 'Error: docker is not installed.' >&2
    exit 1
fi

id -Gn "$USER" | grep -qw 'docker' > /dev/null
if [ $? -eq 0 ] || [ "$(id -u)" -eq '0' ]; then
    DOCKER_CMD="$(command -v docker)"
else
    DOCKER_CMD="sudo $(command -v docker)"
fi

mkdir -p "$MTG_CONFIG" || true

MTG_SECRET="$MTG_CONFIG/secret"
MTG_ENV="$MTG_CONFIG/env"

if [ ! -f "$MTG_ENV" ]; then
    MTG_IMAGENAME="${MTG_IMAGENAME:-nineseconds/mtg:latest}"
    MTG_PORT="${MTG_PORT:-3128}"
    MTG_CONTAINER="${MTG_CONTAINER:-mtg}"

    echo "MTG_IMAGENAME=$MTG_IMAGENAME" > "$MTG_ENV"
    echo "MTG_PORT=$MTG_PORT" >> "$MTG_ENV"
    echo "MTG_CONTAINER=$MTG_CONTAINER" >> "$MTG_ENV"
fi

set -a
source "$MTG_ENV"
set +a

$DOCKER_CMD pull "$MTG_IMAGENAME" > /dev/null
if [ ! -f "$MTG_SECRET" ]; then
    $DOCKER_CMD run \
            --rm \
            "$MTG_IMAGENAME" \
        generate-secret tls -c "$(openssl rand -hex 16).com" \
    > "$MTG_SECRET"
fi

echo "Proxy secret is $(cat "$MTG_SECRET"). Port is $MTG_PORT."

$DOCKER_CMD ps --filter "Name=$MTG_CONTAINER" -aq | xargs -r $DOCKER_CMD rm -fv > /dev/null
$DOCKER_CMD run \
        -d \
        --restart=unless-stopped \
        --name "$MTG_CONTAINER" \
        --ulimit nofile=51200:51200 \
        -p "$MTG_PORT:3128" \
    "$MTG_IMAGENAME" run "$(cat "$MTG_SECRET")" > /dev/null
