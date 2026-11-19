# Ollama providers and secure execution

This branch adds **explicit** local Ollama and **direct** Ollama Cloud support to AutoAR's existing Go AI engine. The provider applies to the CLI agent, scan explanations, AI triage, and the dashboard's shared brain provider path. Ollama selection never silently falls back to OpenCode, OpenRouter, Z.ai, or Gemini.

## Local Ollama on a trusted Tailnet

Set the following in `.env` (or the deployment's secret environment):

```dotenv
AI_PROVIDER=ollama
OLLAMA_BASE_URL=http://100.108.101.105:11434
OLLAMA_MODEL=qwen3:8b
AUTOAR_AI_EXECUTE_COMMANDS=false
AUTOAR_AI_ALLOW_SHELL=false
```

Use the **exact model identifier returned by** `ollama list` on the model host. For Docker-on-macOS/desktop, `http://host.docker.internal:11434` is a common host endpoint; for Linux, Compose supplies `host.docker.internal:host-gateway`. Verify that the Ollama server listens on a Tailnet interface and restricts access at host firewall / Tailnet policy. Do **not** expose port 11434 to the public internet.

Basic connectivity: `curl --fail --silent http://100.108.101.105:11434/api/tags`.

## Ollama Cloud (direct API)

```dotenv
AI_PROVIDER=ollama-cloud
OLLAMA_CLOUD_MODEL=your-available-cloud-model
OLLAMA_API_KEY=put-your-key-in-a-secret-manager-or-local-ignored-env
AUTOAR_AI_EXECUTE_COMMANDS=false
```

AutoAR calls `https://ollama.com/api/chat` directly using `Authorization: Bearer`. This mode **ignores** `OLLAMA_BASE_URL`; it never sends the Ollama Cloud key to a custom local host. Put the exact available cloud model ID in `OLLAMA_CLOUD_MODEL`. No subscription or free-tier claim is made; check your Ollama account quotas.

## Dashboard settings

System → AI Providers exposes: provider selector, local endpoint, local model, cloud model, and masked cloud key. Settings persist in the existing DB-backed settings store and apply to the live process. The public `/api/config` response reports only key *presence*. Protect the database and its backups as credentials may be stored there.

## Security default and threat boundaries

* Inference works with `AUTOAR_AI_EXECUTE_COMMANDS=false`, but **model-generated commands are blocked**. This is intentional: tool output is untrusted and can contain prompt injection.
* To enable AI-proposed AutoAR scanning in an isolated scanner, set `AUTOAR_AI_EXECUTE_COMMANDS=true` **and** `AUTOAR_AI_ALLOWED_DOMAINS=authorized.example.org,other.example.org`. Only explicitly allowlisted AutoAR subcommands/flags are accepted; exact domain and subdomains match. The scope must not be empty.
* Model-generated `run_shell` is **always prohibited**, even if a legacy `AUTOAR_AI_ALLOW_SHELL` value is set. A model cannot execute arbitrary binaries, `curl`, `nmap`, or read arbitrary local paths via `autoar explain`. The AI follow-up triage function returns suggested commands for **manual review only**; it does not execute them. Manual CLI use remains separately authorized by the operator.
* CLI subcommands invoked independently by an operator are outside this AI-command gate; the gate does **not** make the entire scanner safe to run against untrusted targets.
* Default Compose binds the API and PostgreSQL only to localhost, removes the Docker socket, and keeps the image's non-root user. Publish access through an authenticated, TLS-terminating reverse proxy or a restricted Tailnet.
* Existing legacy provider selection is preserved with `AI_PROVIDER=legacy`; no automatic fallback from Ollama is allowed. Legacy provider credentials can still be configured independently.
* No credentials should be placed in a committed file, shell history, logs, or browser localStorage.

## Validation checklist

Run in a clean environment with network access and Go >= the version required by `go.mod`:

```sh
go test ./internal/brain ./internal/api
go test ./...
go vet ./...
docker compose config
```

Test local Ollama and Cloud using non-sensitive sample prompts. Confirm an invalid Cloud key returns a sanitized HTTP error, failed Ollama calls never hit legacy provider endpoints, and AI-generated `run_shell` is blocked by default.

## Remaining security review requirements

Before using AutoAR on sensitive infrastructure, review pinned build dependencies, built image provenance/SBOM, bundled binaries and JavaScript, the database credential encryption model, arbitrary URL fetches/SSRF, scan target authorization, and API authorization on every route. This integration is **not** a malware-free attestation.

## Supply chain and release verification

The remediation branch removes the tracked prebuilt `entrypoint` executable and builds the runtime entrypoint from Go source. The Docker build pins TruffleHog and both wordlist/template repositories to reviewed, explicit Git commit hashes and pins PureDNS to a specific module version. The Docker build no longer mutates `go.mod` using `go get`. The CI security job runs a pinned GoSec version and fails on high-severity, high-confidence findings; it must complete successfully before merge.

**Limitations:** A fixed hash improves repeatability but is not proof of benign content. Upstream templates can contain active requests, including high-impact operations. Review template inputs, isolated scanner permissions, and test against authorized scope. Do not equate a static scan with an independent malware review of all source and dependencies.
