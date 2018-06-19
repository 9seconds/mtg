#!/bin/bash
set -eu -o pipefail

IMAGE_NAME="nineseconds/mtg"
CONTAINER_NAME="mtg"
SECRET_PATH="$HOME/.mtg.secret"
PROXY_PORT=444
STAT_PORT=3129

[[ -e "$SECRET_PATH" ]] || (
  openssl rand -hex 16 > "$SECRET_PATH"
  chmod 0400 "$SECRET_PATH"
)

# docker pull "$IMAGE_NAME"
docker ps --filter "Name=$CONTAINER_NAME" -aq | xargs -r docker rm -fv
docker run \
    --name "$CONTAINER_NAME" \
    --sysctl 'net.ipv4.ip_local_port_range=10000 65000' \
    --sysctl net.ipv4.tcp_congestion_control=bbr \
    --sysctl net.ipv4.tcp_fastopen=3 \
    --sysctl net.ipv4.tcp_fin_timeout=30 \
    --sysctl net.ipv4.tcp_keepalive_time=1200 \
    --sysctl net.ipv4.tcp_max_syn_backlog=4096 \
    --sysctl net.ipv4.tcp_max_tw_buckets=5000 \
    --sysctl net.ipv4.tcp_mtu_probing=1 \
    --sysctl 'net.ipv4.tcp_rmem=4096 87380 67108864' \
    --sysctl net.ipv4.tcp_syncookies=1 \
    --sysctl net.ipv4.tcp_tw_reuse=1 \
    --sysctl 'net.ipv4.tcp_wmem=4096 65536 67108864' \
    --ulimit nofile=51200:51200 \
    --restart=unless-stopped \
    -p $PROXY_PORT:3128 \
    -p $STAT_PORT:3129 \
  "$IMAGE_NAME" "$(cat "$SECRET_PATH")"
