#!/usr/bin/env python3
"""Remove Go port metadata comments from Hollow source files."""

from __future__ import annotations

import re
from pathlib import Path

REPO = Path(__file__).resolve().parents[1]
ROOTS = [REPO / "backend", REPO / "runtime", REPO / "desktop", REPO / "types", REPO / "scripts"]

SKIP_DIRS = {"node_modules", "dist", ".git"}
EXTENSIONS = {".ts", ".tsx", ".cjs", ".d.ts"}

TOP_META = re.compile(
    r"^// (?:PORT STATUS: active|source path: .+|confidence: (?:high|medium|low)|status: phase_[ab]_(?:compile|draft))\s*$"
)
PORT_LINE = re.compile(r"^// PORT: .+\s*$")
DUPLICATE_GO_PATH = re.compile(r"^// backend/.+\.go\s*$")

PORT_BLOCK = re.compile(
    r"\n?/\*\s*\n"
    r"PORT STATUS\s*\n"
    r"(?:.*\n)*?"
    r"\*/\s*\n?$",
    re.MULTILINE,
)

INLINE_REPLACEMENTS = [
    (
        r"// Command Schemas \(mirroring serve\.go types\)",
        "// Command Schemas",
    ),
    (
        r'// Go serve\.go always sends "done" after Prompt returns; mirror that for IPC\.',
        '// Always send "done" after Prompt returns.',
    ),
    (
        r"// Renderer talks IPC only — NO WebSocket, NO Go binary spawn\.",
        "// Renderer talks IPC only — no WebSocket.",
    ),
    (
        r"elapsed: elapsedMs \* 1000000, // Y in nanoseconds to match Go",
        "elapsed: elapsedMs * 1_000_000, // nanoseconds",
    ),
    (
        r"// We run ExecuteSkillManage using Effect\.runSync because ApplySkillPending in Go",
        "// We run ExecuteSkillManage using Effect.runSync because ApplySkillPending",
    ),
    (
        r"// because BuildSkillInvocationMessage has a synchronous signature in Go\.",
        "// because BuildSkillInvocationMessage has a synchronous signature.",
    ),
    (
        r"// Normalize separator to slash like Go's filepath\.ToSlash",
        "// Normalize separator to forward slashes",
    ),
    (
        r"^// PORT: mirrors Flame's packages/coding-agent/src/utils/shell\.ts tracking \+\s*$",
        "",
    ),
]


def should_process(path: Path) -> bool:
    if any(part in SKIP_DIRS for part in path.parts):
        return False
    return path.suffix in EXTENSIONS or path.name.endswith(".d.ts")


def strip_top_metadata(lines: list[str]) -> list[str]:
    while lines and TOP_META.match(lines[0]):
        lines.pop(0)
    while lines and lines[0] == "":
        lines.pop(0)
    return lines


def clean_content(text: str) -> str:
    original = text

    text = PORT_BLOCK.sub("\n", text)

    lines = text.splitlines(keepends=True)
    if lines and lines[0].endswith("\n"):
        bare = [lines[0].rstrip("\n")] + [ln.rstrip("\n") for ln in lines[1:]]
    else:
        bare = [ln.rstrip("\n") for ln in lines]

    changed = True
    while changed and bare:
        changed = False
        if bare and PORT_LINE.match(bare[0]):
            bare.pop(0)
            changed = True
        if bare and DUPLICATE_GO_PATH.match(bare[0]):
            bare.pop(0)
            changed = True
        bare = strip_top_metadata(bare)

    text = "\n".join(bare)
    if original.endswith("\n"):
        text += "\n"

    for pattern, repl in INLINE_REPLACEMENTS:
        text = re.sub(pattern, repl, text, flags=re.MULTILINE)

    # Collapse 3+ consecutive blank lines at file start to one.
    text = re.sub(r"^\n{2,}", "\n", text)
    return text


def main() -> None:
    changed_files: list[Path] = []
    for root in ROOTS:
        if not root.exists():
            continue
        for path in sorted(root.rglob("*")):
            if not path.is_file() or not should_process(path):
                continue
            original = path.read_text(encoding="utf-8")
            cleaned = clean_content(original)
            if cleaned != original:
                path.write_text(cleaned, encoding="utf-8")
                changed_files.append(path)

    # .gitignore
    gitignore = REPO / ".gitignore"
    if gitignore.exists():
        original = gitignore.read_text(encoding="utf-8")
        cleaned = original.replace("# Go tooling (legacy)\n", "")
        if cleaned != original:
            gitignore.write_text(cleaned, encoding="utf-8")
            changed_files.append(gitignore)

    # bundled script with Go mention
    py_path = (
        REPO
        / "backend"
        / "skills"
        / "bundled"
        / "productivity"
        / "google-workspace"
        / "scripts"
        / "_enough_home.py"
    )
    if py_path.exists():
        original = py_path.read_text(encoding="utf-8")
        cleaned = original.replace("without importing the Go binary.", "without a separate agent binary.")
        if cleaned != original:
            py_path.write_text(cleaned, encoding="utf-8")
            changed_files.append(py_path)

    print(f"Updated {len(changed_files)} files")
    for path in changed_files[:20]:
        print(f"  {path.relative_to(REPO)}")
    if len(changed_files) > 20:
        print(f"  ... and {len(changed_files) - 20} more")


if __name__ == "__main__":
    main()
