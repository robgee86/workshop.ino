# workshop.ino

A live, offline-first **lab handbook** for hands-on workshops. Authors write the workshop as a folder of Markdown files; participants follow along step by step in a browser. Code blocks have copy buttons, diffs between steps are pretty-printed, Mermaid diagrams render, and progress is tracked in each participant's browser.

It runs on the participant's own device — an ARM64 SBC such as the **Arduino Uno Q** or **Arduino Ventuno Q** — and applying a step's **solution** writes straight to the Arduino app on that same device (under `/home/arduino/ArduinoApps`), so a click in the browser changes the code the participant is building.

> Content is read from disk on every request, so an author can fix a typo
> mid-workshop and have the change show up on refresh.

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
  workshop.ino  ·  serving /…/workshop.ino/content

    http://localhost:8080
    http://192.168.1.86:8080
```

Open `http://localhost:8080` in a browser on the device. (The other URLs let you reach it from your laptop while authoring.)

### Build a distributable binary

```bash
go build -o workshop.ino .
./workshop.ino -content ./my-workshop -addr :8080
```

### Flags

| Flag       | Default                      | What it does                                          |
|------------|------------------------------|-------------------------------------------------------|
| `-content` | `./content`                  | Path to your workshop folder (resolved against CWD).  |
| `-addr`    | `:8080`                      | Address to listen on (`:9000`, `127.0.0.1:8080`, …).  |
| `-apps`    | `~/ArduinoApps`              | Directory holding the Arduino apps a step's solution is applied to. Resolves to `$HOME/ArduinoApps`, falling back to `/home/arduino/ArduinoApps` when `$HOME` is unset (e.g. in the container). Point it at a scratch dir when testing on a dev machine. |

`-content` is resolved against the current working directory, not the binary's location — `cd` into the folder that holds `content/` before running.

---

## Deployment

workshop.ino is meant to run **on the participant's device** (the SBC). Two ways to install it:

> **Applying a solution needs write access to the apps folder** (`/home/arduino/ArduinoApps`). The native binary has everything out of the box, whereas the sample Compose file bind-mounts that host folder into the container read-write.

### Native binary (+ optional systemd)

1. Grab `workshop.ino-linux-arm64` from the [latest GitHub Release](https://github.com/arduino/workshop.ino/releases/latest).
2. Drop it in any directory with a `content/` folder beside it:

   ```
   anywhere/
     workshop.ino-linux-arm64    (the binary, rename if you like)
     content/                    (your workshop contents)
   ```

3. Run it:

   ```bash
   chmod +x workshop.ino-linux-arm64
   ./workshop.ino-linux-arm64
   ```

   The default `-content ./content` finds the folder next to it; the handbook is on `http://<host-ip>:8080/`.

For a managed service, point [`deploy/workshop.ino.service`](deploy/workshop.ino.service) at wherever you dropped the binary (edit `WorkingDirectory` and `ExecStart`), then:

```bash
sudo cp deploy/workshop.ino.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now workshop.ino
```

### Docker container + Compose

1. Copy [`deploy/docker-compose.yml`](deploy/docker-compose.yml) to the board.
2. Put a `content/` folder next to it.
3. `docker compose up -d`.

The handbook is then on `http://<board-ip>:8080/` (host `:8080` → container `:8080`, so it runs fine without root or a `sysctl` tweak — including under rootless Docker). The image is pulled from GHCR — refresh with `docker compose pull && docker compose up -d`. The Compose file mounts `/home/arduino/ArduinoApps` read-write so Apply works; for applied files to be owned by the arduino user rather than root, run the container as that user (see the comment in the file).

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
