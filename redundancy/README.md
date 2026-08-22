# Direct Sportybet Integration (Bypassing Cloudflare Workers)

This folder contains a direct Go implementation that hits Sportybet's API without going through the Cloudflare Worker proxy (`live-events.aytholu.workers.dev`). 

## Why does this exist?
As discovered in testing, HuggingFace Spaces IP addresses are aggressively blocked/tarpitted by Cloudflare's Bot Management when trying to hit `.workers.dev` subdomains. This causes 10-30 second timeouts on HF, while it works instantly on Local or Railway. 

However, Sportybet itself *does not block HF Space IPs*, meaning the Go backend can bypass Cloudflare and fetch the tickets and live match results perfectly fine in < 100ms.

## How to switch to this implementation
If you ever need to abandon the Cloudflare Worker and switch to direct fetching:

1. Copy the contents of `redundancy/direct_client.go` and completely overwrite `internal/worker/cloudflare.go`.
2. Rename `directClientImpl` to `cloudflareClientImpl` to preserve the interface, OR modify `cmd/server/main.go` to use `worker.NewDirectClient()` instead of `worker.NewCloudflareClient()`.
3. Rebuild and deploy the server.
