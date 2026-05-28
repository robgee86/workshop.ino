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
