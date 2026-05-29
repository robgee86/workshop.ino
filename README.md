# workshopify

A live, offline-first **lab handbook server** for hands-on workshops. Authors write the workshop as a folder of Markdown files; participants open a link on their own laptop and follow along step by step. Code blocks have copy buttons, diffs between steps are pretty-printed, Mermaid diagrams render, and progress is tracked in each participant's browser.

> Content is read from disk on every request, so an instructor can fix a typo
> mid-workshop and have the change show up on refresh..

**📖 [Authoring guide → DOCS.md](DOCS.md)** — the full reference for the
`content/` folder, frontmatter, side quests, attachments, patches, and the playbook for turning an event agenda into a workshop.


## Quick start

From the repository root:

```bash
go run . -content ./content
```

The example Arduino-blink workshop in [content/](content/) will start serving.
You'll see something like:

```
  workshopify  ·  serving /…/workshopify/content

    http://localhost:8080
    http://192.168.1.86:8080
```

Share one of the LAN URLs in the room. Participants open it in any browser — no install on their side.

### Build a distributable binary

```bash
go build -o workshopify .
./workshopify -content ./my-workshop -addr :8080
```

### Flags

| Flag       | Default     | What it does                                          |
|------------|-------------|-------------------------------------------------------|
| `-content` | `./content` | Path to your workshop folder (resolved against CWD).  |
| `-addr`    | `:8080`     | Address to listen on (`:9000`, `127.0.0.1:8080`, …).  |

`-content` is resolved against the current working directory, not the binary's location — `cd` into the folder that holds `content/` before running.

---

## Deployment

Two ways to run workshopify on a board (or any Linux host). Pick the one that fits.

### Native binary (+ optional systemd)

1. Grab `workshopify-linux-arm64` from the [latest GitHub Release](https://github.com/arduino/workshopify/releases/latest).
2. Drop it in any directory with a `content/` folder beside it:

   ```
   anywhere/
     workshopify-linux-arm64    (the binary; rename to `workshopify` if you like)
     content/                   (your workshop)
   ```

3. Run it:

   ```bash
   chmod +x workshopify-linux-arm64
   ./workshopify-linux-arm64
   ```

   The default `-content ./content` finds the folder next to it; the handbook is on `http://<host-ip>:8080/`.

For a managed service, point [`deploy/workshopify.service`](deploy/workshopify.service) at wherever you dropped the binary (edit `WorkingDirectory` and `ExecStart`), then:

```bash
sudo cp deploy/workshopify.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now workshopify
```

### Docker container + Compose

1. Copy [`deploy/docker-compose.yml`](deploy/docker-compose.yml) to the board.
2. Put a `content/` folder next to it.
3. `docker compose up -d`.

The handbook is then on `http://<board-ip>/` (host `:80` → container `:8080`). The image is pulled from GHCR — refresh with `docker compose pull && docker compose up -d`.

Both paths use the same arm64 artifacts published by the release workflow ([`.github/workflows/release.yml`](.github/workflows/release.yml)) on every `v*` tag.

---

## Development

### Tests

```bash
go test ./...
```

### Repository layout

- [main.go](main.go) — CLI flags, LAN URL printing, server startup.
- [internal/content/](internal/content/) — pure logic:
  - [render.go](internal/content/render.go) — YAML frontmatter parsing.
  - [scan.go](internal/content/scan.go) — folder walking, ordering, side-quest discovery.
  - [markdown.go](internal/content/markdown.go) — Markdown rendering (goldmark) with chroma highlighting, Mermaid passthrough, relative-image rewriting.
  - [patch.go](internal/content/patch.go) — unified-diff parsing into `DiffFile`/`DiffHunk`/`DiffLine`.
- [internal/server/](internal/server/) — HTTP layer:
  - [handlers.go](internal/server/handlers.go) — index, step, side-quest, download routes; view-model construction.
  - [download.go](internal/server/download.go) — path-traversal guard for `/dl/…`.
  - [templates/](internal/server/templates/) — embedded `html/template` pages.
  - [assets/](internal/server/assets/) — embedded CSS, JS (Alpine, Mermaid), logo, favicon.
- [content/](content/) — the example Arduino-blink workshop (runs out of the
  box, exercises every feature).
- [DOCS.md](DOCS.md) — authoring guide for workshop content.

### Live editing while developing

Content under [content/](content/) is read from disk on every request, so edits to `.md` / `.patch` / asset files show up on refresh without restarting. The templates, CSS and JS *are* embedded into the binary via `go:embed`, so changes to those need a rebuild (`go run .` again, or `go build`).
