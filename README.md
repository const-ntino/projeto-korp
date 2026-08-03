# Projeto Korp

Serviço HTTP em Go, containerizado e provisionado de ponta a ponta com Ansible, com proxy reverso NGINX e observabilidade via Prometheus e Grafana.

Desenvolvido como desafio técnico do processo seletivo da Korp.

## Arquitetura

```
                        ┌────────────────────────────────────────┐
                        │        rede bridge: projeto-korp-net   │
                        │                                        │
  client                │  ┌───────┐        ┌──────────────────┐ │
  curl :80  ───────────►│  │ nginx │───────►│ http-server-     │ │
                        │  │  :80  │  :8080 │projeto-korp :8080│ │
                        │  └───────┘        └──────────┬───────┘ │
                        │                              │ /metrics│
                        │  ┌────────────┐   scrape     │         │
                        │  │ prometheus │◄─────────────┘         │
                        │  │   :9090    │                        │
                        │  └─────┬──────┘                        │
                        │        │ datasource                    │
                        │  ┌─────▼──────┐                        │
                        │  │  grafana   │                        │
                        │  │   :3000    │                        │
                        │  └────────────┘                        │
                        └────────────────────────────────────────┘
```

Apenas o NGINX expõe porta ao host (80). O serviço Go, Prometheus e Grafana são acessíveis somente dentro da rede Docker, comunicando-se por nome de serviço.

## Stack

| Componente | Imagem/versão |
|---|---|
| Serviço HTTP | Go 1.22, build multi-stage sobre `distroless/static-debian12:nonroot` |
| Proxy reverso | `nginx:1.27-alpine` |
| Métricas | `prom/prometheus:v2.54.1` |
| Dashboards | `grafana/grafana:11.2.0` |
| Automação | Ansible + coleção `community.docker` |

## Como rodar

### Opção 1: Ansible (provisionamento completo, um único comando)

Requisitos: máquina Linux com Python 3, Ansible instalado e acesso sudo.

```bash
cd ansible
ansible-galaxy collection install -r requirements.yml
ansible-playbook -i inventory.ini playbook.yml
```

O playbook instala o Docker, copia o projeto para `/opt/projeto-korp`, builda a imagem do serviço, cria a rede bridge, sobe a stack via Docker Compose e valida o serviço fazendo uma requisição HTTP real, exibindo a resposta no console ao final da execução.

Por padrão o inventário aponta para `localhost` (`ansible_connection=local`). Para provisionar uma máquina remota, edite `ansible/inventory.ini` com o host de destino e as credenciais SSH.

### Opção 2: Docker Compose direto

```bash
docker compose up -d --build
curl http://localhost:80/projeto-korp
```

### Script de validação local

`scripts/validate.sh` roda os testes Go, sobe a stack, valida o endpoint principal, checa a presença das métricas esperadas, confirma que o Prometheus está com o target `up` e verifica a saúde do Grafana:

```bash
./scripts/validate.sh
```

## Endpoints do serviço

| Rota | Descrição |
|---|---|
| `GET /projeto-korp` | Retorna `{"nome": "Projeto Korp", "horario": "<UTC RFC3339>"}` |
| `GET /healthz` | Health check simples, `200 ok` |
| `GET /metrics` | Métricas no formato Prometheus |

O horário é resolvido a cada requisição via `time.Now().UTC()`, injetado como dependência (`func() time.Time`) para permitir testes determinísticos sem mockar o relógio do sistema.

## Métricas

Implementadas com `prometheus/client_golang`, em um registry próprio (não o `DefaultRegisterer` global):

- **`http_server_projeto_korp_up`** (gauge): disponibilidade do serviço.
- **`http_server_projeto_korp_requests_total`** (counter, labels `method` e `status_code`): volume de requisições.

## Dashboard Grafana

Provisionado automaticamente, sem nenhum passo manual: o Grafana sobe já com o datasource do Prometheus (`grafana/provisioning/datasources/datasources.yml`) e o dashboard (`grafana/provisioning/dashboards/dashboards.yml` + `grafana/dashboards/http-server-projeto-korp-dashboard.json`) carregados.

Acesso: `http://localhost:3000` (`admin` / `admin`, troca de senha solicitada no primeiro login).

Painéis:
- **Disponibilidade**: estado atual do gauge `up`.
- **Volume de requisições por método e status**: série temporal de `http_server_projeto_korp_requests_total`, agrupada por `method` e `status_code`.

## Automação Ansible

Estruturado em roles, cada uma com uma responsabilidade única:

- **`docker`**: instala Docker Engine, Compose plugin e dependências.
- **`app_image`**: copia o projeto para o host de destino e builda a imagem via módulo `docker_image` (não shell).
- **`deployment`**: cria a rede bridge via `docker_network` e sobe a stack via `docker_compose_v2`.
- **`monitoring_validation`**: aguarda o serviço responder (retry com backoff) e imprime o corpo da resposta HTTP no console via `debug`.

Variáveis centralizadas em `ansible/group_vars/projeto_korp.yml` (diretório de deploy, nome da rede, nome da imagem, URL de validação), evitando valores hardcoded espalhados pelas tasks.

## Testes

```bash
go test ./...
```

Cobertura em `internal/server/server_test.go`: resposta e formato do endpoint principal com relógio fixo, rejeição de métodos não-GET, health check, e presença das métricas de disponibilidade e volume por método/status.

## Estrutura do repositório

```
.
├── ansible/                  # provisionamento completo (roles + playbook)
├── cmd/http-server/          # entrypoint da aplicação
├── internal/server/          # handler HTTP + métricas + testes
├── nginx/conf.d/             # configuração do proxy reverso
├── prometheus/               # configuração de scrape
├── grafana/                  # provisioning (datasource + dashboard) e dashboard JSON
├── scripts/validate.sh       # validação local end-to-end
├── Dockerfile                # build multi-stage, imagem final distroless non-root
└── docker-compose.yml
```

## Decisões técnicas

**Imagem distroless non-root.** O build multi-stage compila um binário estático (`CGO_ENABLED=0`) e a imagem final não tem shell, gerenciador de pacotes nem usuário root, reduzindo superfície de ataque e tamanho de imagem.

**Registry Prometheus isolado.** As métricas usam um `prometheus.NewRegistry()` próprio em vez do registrador global padrão, evitando poluição do endpoint `/metrics` com métricas internas do runtime Go que não fazem parte do escopo do desafio.

**Ansible com módulos idempotentes, não shell.** `docker_image`, `docker_network` e `docker_compose_v2` do `community.docker` garantem que rodar o playbook múltiplas vezes não produza efeitos colaterais (`changed=0` na segunda execução), em vez de uma sequência de comandos shell que precisaria de checks manuais de idempotência.

**Rede criada explicitamente antes do Compose.** A rede bridge é criada pela role `deployment` antes de subir a stack, e o `docker-compose.yml` referencia essa mesma rede pelo nome, atendendo ao requisito de rede criada separadamente da criação dos containers.