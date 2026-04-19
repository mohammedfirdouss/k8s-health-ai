#!/usr/bin/env python3
"""Resolve repo-relative go build target from Cursor hook JSON (stdin). Prints ./pkg or empty.

PAYLOAD ASSUMPTION: Cursor afterFileEdit hook sends JSON with one of the keys
"file_path", "path", "uri", or "file" containing the edited file's absolute path.
If the schema changes, this script silently prints nothing and the caller falls
back to `go build ./...`.
"""
import json
import os
import subprocess
import sys


def main() -> None:
    root = os.path.normpath(sys.argv[1])
    raw = sys.stdin.read()
    path = ""
    try:
        d = json.loads(raw) if raw.strip() else {}
    except json.JSONDecodeError:
        d = {}
    for key in ("file_path", "path", "uri", "file"):
        v = d.get(key)
        if isinstance(v, str) and v.endswith(".go"):
            path = os.path.normpath(v)
            break
    if not path:
        return
    if not (path.startswith(root + os.sep) or path == root):
        return
    dirpath = os.path.dirname(path)
    cur = dirpath
    while cur.startswith(root + os.sep) or cur == root:
        try:
            out = subprocess.check_output(
                ["go", "list", "-e", "-f", "{{.ImportPath}}", cur],
                cwd=root,
                stderr=subprocess.DEVNULL,
                text=True,
            ).strip()
            if out and "command-line-arguments" not in out:
                rel = os.path.relpath(cur, root)
                print("." if rel == "." else "./" + rel.replace(os.sep, "/"), end="")
                return
        except subprocess.CalledProcessError:
            pass
        parent = os.path.dirname(cur)
        if parent == cur:
            break
        cur = parent


if __name__ == "__main__":
    main()
