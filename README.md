# faker-mcp

A free, open-source MCP server that generates fake data using [faker-go](https://github.com/jaswdr/faker). Powered by [mcp-grpc-gateway](https://github.com/cadenya/mcp-grpc-gateway).

**Hosted endpoint:** `https://free.cadenya.com/faker-mcp`

## Why?

This is more of a proof of concept for [mcp-grpc-gateway](https://github.com/cadenya/mcp-grpc-gateway) than it is a valuable MCP server. However there are times when using fake data in an LLMs conversation can be useful to generate company names, addresses, etc. Can LLMs do that? Sure. But that's not the point.

It is hosted for free by [Cadenya](https://cadenya.com), an Agent Runtime.

## Tools

The footprint of tools is very low (not every faker category is loaded into context, that would be a waste of input tokens).

| Tool | Description |
|------|-------------|
| `GetFakerOptions` | Lists supported fake data generators. Use `filter` to narrow by name, category, or description. |
| `GenerateFake` | Generates one fake data value by option name and optional arguments. |

The service builds its catalog from the exported faker-go API at startup. It includes every reflected method that can be called with JSON-friendly scalar arguments and returned as text or JSON. Example faker names: `person.name`, `internet.email`, `company.name`, `address.city`, `lorem.sentence`, `faker.int_between`, `uuid.v4`, `color.hex`.

## Usage

### Claude Code

```bash
claude mcp add faker --transport http https://free.cadenya.com/faker-mcp
```

### Codex

```bash
codex mcp add faker --url https://free.cadenya.com/faker-mcp
```

### Claude Desktop

Add to your MCP configuration:

```json
{
  "mcpServers": {
    "faker": {
      "type": "streamable-http",
      "url": "https://free.cadenya.com/faker-mcp"
    }
  }
}
```

### curl

```bash
# Initialize
curl -sS https://free.cadenya.com/faker-mcp \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"curl","version":"1.0"}}}'

# List tools
curl -sS https://free.cadenya.com/faker-mcp \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'

# Generate fake data
curl -sS https://free.cadenya.com/faker-mcp \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"GenerateFake","arguments":{"name":"internet.email"}}}'
```

## Run Locally

```bash
docker compose up --build
```

The MCP endpoint is available at `http://localhost:8080/faker-mcp`.

## Development

### Prerequisites

- Go 1.24+
- [buf](https://buf.build/) (for proto regeneration)
- Docker (for local testing)

### Build

```bash
go build -o faker-mcp .
```

### Test

```bash
go test ./...
```

### Regenerate Protos

```bash
buf generate
```

## Deployment

This project uses [Kustomize](https://kustomize.io/) manifests designed for [ArgoCD](https://argo-cd.readthedocs.io/):

```bash
# Preview production manifests
kubectl kustomize k8s/overlays/production

# Apply directly (prefer ArgoCD for production)
kubectl apply -k k8s/overlays/production
```

The production overlay deploys to `free.cadenya.com/faker-mcp` with 2 replicas.

## Architecture

```
┌─────────────────────────────────────────────┐
│ Pod                                         │
│                                             │
│  ┌──────────────┐    ┌───────────────────┐  │
│  │ faker-mcp    │    │ mcp-grpc-gateway  │  │
│  │ (gRPC :50051)│◄───│ (HTTP :8080)      │  │
│  └──────────────┘    └───────────────────┘  │
│                              │              │
└──────────────────────────────┼──────────────┘
                               │
                         Ingress (free.cadenya.com/faker-mcp)
```

The faker gRPC service and the MCP gateway run as a sidecar pair. The gateway uses gRPC reflection to discover the service schema and exposes it as MCP tools over HTTP.

## License

[MIT](LICENSE)
