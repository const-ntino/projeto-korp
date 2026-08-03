# Projeto Korp

Servico HTTP em Go para o desafio tecnico Projeto Korp. O endpoint `GET /projeto-korp` retorna o nome do projeto e o horario atual em UTC.

## Arquitetura

```text
Host
  |
  | http://localhost:80/projeto-korp
  v
Nginx :80
  |
  | rede Docker projeto-korp-net
  v
Go HTTP service :8080
  |
  | /metrics
  v
Prometheus :9090 <---- Grafana :3000
```

O container do servico Go nao publica portas no host. Nginx e a unica entrada HTTP externa. Prometheus e Grafana ficam na mesma rede Docker para acessar o servico e um ao outro.

## Requisitos

- Docker
- Docker Compose v2
- Go 1.22+ para testes locais
- Ansible para provisionamento automatizado
- Collection Ansible `community.docker`

## Execucao Local

```bash
docker compose up -d --build
curl http://localhost:80/projeto-korp
```

Resposta esperada:

```json
{"nome":"Projeto Korp","horario":"2026-08-03T15:34:56Z"}
```

O valor de `horario` muda a cada chamada e usa UTC no formato RFC3339.

## Observabilidade

- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000
- Grafana login: `admin` / `admin`
- Dashboard: `HTTP Server Projeto Korp`

Metricas expostas pelo servico:

- `http_server_projeto_korp_up`
- `http_server_projeto_korp_requests_total{method,status_code}`

A metrica `http_server_projeto_korp_up` e uma gauge exposta pelo proprio processo. Ela atende ao desafio, mas tem uma limitacao: se o container cair, o endpoint `/metrics` deixa de responder e a gauge nao consegue reportar `0`. Em producao, uma sonda externa com `blackbox_exporter` mede a disponibilidade HTTP de fora do processo.

## Docker E Nginx

O `Dockerfile` usa multi-stage build:

- `golang:1.22-alpine` compila o binario com `CGO_ENABLED=0`.
- `gcr.io/distroless/static-debian12:nonroot` roda a aplicacao.
- A imagem final copia apenas o binario.
- O processo roda como `nonroot:nonroot`.

O Nginx configura headers de proxy reverso:

- `Host`
- `X-Real-IP`
- `X-Forwarded-For`
- `X-Forwarded-Proto`

Tambem define timeouts explicitos para conexao, envio, leitura e resposta ao cliente.

## Provisionamento Com Ansible

Instale a collection:

```bash
ansible-galaxy collection install -r ansible/requirements.yml
```

Execute o playbook:

```bash
ansible-playbook -i ansible/inventory.ini ansible/playbook.yml
```

O playbook usa roles:

- `docker`: instala Docker, Compose v2, SDK Python Docker e inicia o servico.
- `app_image`: copia o projeto para `/opt/projeto-korp` e cria a imagem com `community.docker.docker_image`.
- `deployment`: cria a rede com `community.docker.docker_network` e sobe a stack com `community.docker.docker_compose_v2`.
- `monitoring_validation`: valida `GET /projeto-korp` com `ansible.builtin.uri` e imprime o corpo da resposta.

Para provar idempotencia, rode o playbook duas vezes. A segunda execucao deve terminar sem mudancas fora de verificacoes inevitaveis do ambiente.

## Validacao

```bash
go test ./...
docker compose config
bash scripts/validate.sh
ansible-playbook -i ansible/inventory.ini ansible/playbook.yml --syntax-check
```

Ferramentas de lint recomendadas:

```bash
golangci-lint run
hadolint Dockerfile
ansible-lint ansible/playbook.yml
```
# projeto-korp
