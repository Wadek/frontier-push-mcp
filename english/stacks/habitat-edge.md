# Stack: Habitat edge (waka-net / cloudflared)

**Learned while launching Pigeon** (`pigeon.wakalabs.net`) and auditing the wakalabs habitat.

## Detect

- `cloudflared/config.yml` with `ingress:` hostnames  
- External Docker network `waka-net`  
- Optional `scripts/manage-tunnel-routes.ps1`  

## Standard publish flow (new public app)

```text
1. App joins waka-net; prefer NO host ports (like tasks/books)
2. App has its own auth (or goes through gateway oauth2-proxy)
3. Edit cloudflared/config.yml — add hostname + service
4. Publish DNS:
   cloudflared tunnel route dns --overwrite-dns <tunnel-id> <hostname>
   (or fixed manage-tunnel-routes.ps1 once CLI args are correct)
5. docker compose restart cloudflared-waka
6. Probe: https://<hostname>/api/health (or /)
7. Frontier: learn | guard on the app repo before inviting users
```

## Exposure audit (before first public hostname)

Read-only checklist (see also Pigeon `AUDIT-2026-08-19.md`):

| Check | Why |
|-------|-----|
| `docker ps` Ports column | Spot `0.0.0.0` binds (LAN; dangerous if WAN-forwarded) |
| cloudflared ingress list | What is already on the public Internet |
| Direct vs gateway | `home`/`vault`/`immich` style direct routes need strong app auth |
| Router port-forwards | Human confirms — agents cannot see ISP NAT |
| Cookie `Secure` | `Secure=true` breaks `http://127.0.0.1`; use env `COOKIE_SECURE` |

## Pitfalls learned

| Pitfall | Fix |
|---------|-----|
| `express-session` `secure: NODE_ENV===production` | Blocks sessions on localhost HTTP after register — loft never loads |
| manage-tunnel-routes wrong CLI order | Use `cloudflared tunnel route dns --overwrite-dns <id> <hostname>` |
| Config path still `D:\waka-net\...` in old scripts | Use `D:\wakalabs\waka-net\...` after habitat move |
| Committing sqlite under `data/` | gitignore runtime data |
| Local folder still named `frontier-push-mcp` | Rename to `frontier-ship` to match GitHub/module |

## Pigeon reference

- Repo: `github.com/Wadek/pigeon`  
- Internal: `pigeon-web:80`, `pigeon-api:8787` on `waka-net`  
- Public: `https://pigeon.wakalabs.net` → `http://pigeon-web:80`  
- Loopback tryout: `127.0.0.1:8790` only  

## Guard / Optimize

- Run `frontier learn` + `frontier guard` on the app before enabling the tunnel.  
- Add stack playbook link in `.frontier/optimize/stack.md` when Optimize starts (likely `node-express` + this edge playbook).
