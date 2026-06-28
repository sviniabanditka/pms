#!/bin/sh
# Одноразовый bootstrap Garage с ХОСТА (образ garage distroless — без shell внутри).
# Запуск: make s3-init  (после docker compose up -d garage). Идемпотентно.
set -e
G="docker compose exec -T garage /garage -c /etc/garage.toml"

ACCESS_KEY="GK00112233445566778899aabb"
SECRET_KEY="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
BUCKET="pms"

echo "garage-init: waiting for node…"
i=0
until $G status >/dev/null 2>&1; do
  i=$((i+1)); [ "$i" -gt 60 ] && { echo "garage not reachable"; exit 1; }
  sleep 1
done

NID=$($G node id -q 2>/dev/null | cut -d@ -f1)
echo "garage-init: node $NID"

if ! $G layout show 2>/dev/null | grep -q "$NID"; then
  $G layout assign -z dc1 -c 1G "$NID" || true
  $G layout apply --version 1 || true
fi

$G key import --yes -n pms "$ACCESS_KEY" "$SECRET_KEY" 2>/dev/null || echo "key exists"
$G bucket create "$BUCKET" 2>/dev/null || echo "bucket exists"
$G bucket allow --read --write --owner "$BUCKET" --key "$ACCESS_KEY" >/dev/null || true

echo "garage-init: done. bucket=$BUCKET key=$ACCESS_KEY"
