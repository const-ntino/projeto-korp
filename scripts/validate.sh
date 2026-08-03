#!/usr/bin/env bash
set -euo pipefail

go test ./...

docker compose up -d --build

echo "HTTP response:"
curl -fsS http://localhost:80/projeto-korp
echo

echo "Metrics sample:"
curl -fsS http://localhost:80/projeto-korp >/dev/null
curl -fsS http://localhost:80/metrics | grep -E 'http_server_projeto_korp_up|http_server_projeto_korp_requests_total\{method="GET",status_code="200"\}'

echo "Prometheus targets:"
curl -fsS http://localhost:9090/api/v1/targets | grep -q 'http-server-projeto-korp:8080'

echo "Grafana health:"
curl -fsS http://localhost:3000/api/health
echo
