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
#   curl -sfL https://lalala | bash

set -eu -o pipefail

export XDG_CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config}"
export MTG_CONFIG="${MTG_CONFIG:-$XDG_CONFIG_HOME/mtg}"

mkdir -p "$MTG_CONFIG" || true

MTG_SECRET="$MTG_CONFIG/secret"
MTG_ENV="$MTG_CONFIG/env"

if [ ! -f "$MTG_ENV" ]; then
    MTG_IMAGENAME="${MTG_IMAGENAME:-nineseconds/mtg:latest}"
    MTG_PORT="${MTG_PORT:-3128}"
    MTG_CONTAINER="${MTG_CONTAINER:-mtg}"

    echo "MTG_IMAGENAME=${MTG_IMAGENAME}" > "$MTG_ENV"
    echo "MTG_PORT=${MTG_PORT}" >> "$MTG_ENV"
    echo "MTG_CONTAINER=${MTG_CONTAINER}" >> "$MTG_ENV"
fi

set -a
source "$MTG_ENV"
set +a

docker pull "$MTG_IMAGENAME"
if [ ! -f "$MTG_SECRET" ]; then
    docker run \
            --rm \
            "$MTG_IMAGENAME" \
        generate-secret tls -c "$(openssl rand -hex 16).com" \
    > "$MTG_SECRET"
fi

echo "Proxy secret is $(cat "$MTG_SECRET")"

docker ps --filter "Name=$MTG_CONTAINER" -aq | xargs -r docker rm -fv
docker run \
        -d \
        --restart=unless-stopped \
        --name "$MTG_CONTAINER" \
        --ulimit nofile=51200:51200 \
        -p "$MTG_PORT:3128" \
    "$MTG_IMAGENAME" run "$(cat "$MTG_SECRET")"
