# Enough

Enough is a terminal coding agent for inspecting, editing, testing, and iterating on software projects with native tools and parallel subagents. It keeps a persistent session transcript, uses the configured OpenCode-compatible model endpoint, and exposes coding tools directly to the model so changes can be made and verified in the workspace.

## Build

```sh
make build
```

The binary is written to `bin/enough`.

```sh
make install
```

Installs the built binary according to the project `Makefile`.

## Configuration

Enough stores config and runtime data under `~/.enough/` (override with `ENOUGH_HOME`):

```text
~/.enough/config.json          # settings
~/.enough/skills/              # global skills library
~/.enough/agent/sessions/      # session transcripts
```

On first run, Enough migrates an existing `~/.config/enough/config.json` into `~/.enough/config.json`.

Defaults live in `backend/config/config.go`:

- endpoint: `https://opencode.ai/zen/go/v1`
- model: `deepseek-v4-flash`

The API key is stored via the secrets backend, not written to `config.json`.

## Skills

Skills are reusable procedural instructions (Markdown with YAML frontmatter). Enough discovers them from:

- `~/.enough/skills/` — global library (`skill_manage` writes here)
- `{project}/.enough/skills/` — project skills (win name collisions)
- `~/.cursor/skills/` and `{project}/.cursor/skills/` — Cursor-compatible paths

Use `/skills` in the TUI to list discovered skills, `/skill:<name> <args>` to run one, and `/skill-archive <name>` or `/skill-restore <name>` to manage the global library. See `backend/skills/enough_skill/SKILL.md` for the full slash-command reference.

## Tools

Enough exposes native coding tools including:

- `read_file`: reads a file and reports the full line count in the header, even when output is truncated.
- `write_file`: writes a complete file.
- `edit_file`: replaces text in an existing file.
- `list_dir`, `glob`, `grep`: inspect the workspace.
- `bash`: runs shell commands in the workspace.
- `web_search`: searches current web content.
- `agent_swarm`: runs parallel subagents.

### agent_swarm

`agent_swarm` accepts either a `tasks` array or a `goal`.

Each task has:

- `id`: optional display label.
- `prompt`: the complete self-contained worker instruction.
- `depends_on`: optional task ids from the same call. A dependent task starts after those tasks finish and receives their outputs.

Useful options:

- `shared_context`: prepended to every worker prompt.
- `max_concurrency`: number of workers to run at once. Default: `8`.
- `retry`: worker retry count for stream/API errors or empty output. Default: `3`.
- `max_turns_per_agent`: optional cap; workers otherwise run to completion.
- `isolate: "worktree"`: runs each worker in a separate git worktree/branch. Clean worktrees are removed; dirty worktrees are reported and left for review.

Nested swarms are available up to depth `3`. Worktree-isolated workers disable nested `agent_swarm` so nested edits cannot escape the isolated worktree. Without worktree isolation, a swarm call rejects tasks that appear to target the same path; split the work or use `isolate: "worktree"`.

## Known Limits

- Worktree isolation requires a git repository. Outside git, `isolate: "worktree"` falls back to the shared working directory.
- `agent_swarm` does not implement future Flame extras such as consensus, verifier, blackboard, quorum, or transcript saving.
- Parallel editing of the same file in one non-isolated swarm call is blocked rather than merged automatically.

## Tests

```sh
go test -race ./...
```
