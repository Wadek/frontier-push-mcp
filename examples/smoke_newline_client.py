#!/usr/bin/env python3
"""Tiny local smoke client (newline JSON). Not full MCP framing — for quick tool logic checks."""
import json, os, subprocess, sys

def main():
    bin_path = sys.argv[1] if len(sys.argv) > 1 else "./frontier-mcp"
    repo = os.environ.get("FRONTIER_REPO", ".")
    env = os.environ.copy()
    env["FRONTIER_REPO"] = repo
    p = subprocess.Popen(
        [bin_path],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=env,
    )
    # This binary expects Content-Length; use the Go test instead for CI.
    print("Use: go test ./...  and  go run ./cmd/frontier-mcp with an MCP host.", file=sys.stderr)
    p.kill()

if __name__ == "__main__":
    main()
