# Navigate

Open a new tab and navigate it to a URL, or reuse a tab when a tab ID is provided.

```bash
curl -X POST http://localhost:9867/navigate \
  -H "Content-Type: application/json" \
  -d '{"url":"https://pinchtab.com"}'
# CLI Alternative
pinchtab nav https://pinchtab.com
# Response (default is tab ID; use --json for full JSON)
8f9c7d4e1234567890abcdef12345678
```

## CLI Flags

| Flag | Description |
|------|-------------|
| `--tab` | Reuse existing tab |
| `--new-tab` | Force new tab |
| `--block-images` | Block image loading |
| `--block-ads` | Block ads |
| `--snap` | Output snapshot after navigation |
| `--snap-diff` | Output snapshot diff after navigation |
| `--print-tab-id` | Print only tab ID (auto when piped) |
| `--json` | Full JSON response |

## Examples

```bash
pinchtab nav https://example.com              # Navigate, print tab ID
pinchtab nav https://example.com --snap       # Navigate and snapshot
TAB=$(pinchtab nav https://example.com)       # Capture tab ID for reuse
pinchtab nav https://other.com --tab "$TAB"   # Reuse tab
pinchtab nav https://example.com --block-images  # Skip images
```

## API Body Fields

| Field | Description |
|-------|-------------|
| `url` | Target URL (required) |
| `tabId` | Reuse existing tab |
| `newTab` | Force new tab |
| `blockImages` | Block image loading |
| `blockAds` | Block ads |
| `timeout` | Navigation timeout |
| `waitFor` | Wait condition |
| `waitSelector` | Wait for selector |

## Behavior

- Top-level `POST /navigate` and `pinchtab nav <url>` open a new tab when no `tabId` or `--tab` is provided.
- `POST /tabs/{id}/navigate`, `POST /navigate` with `tabId`, and `pinchtab nav <url> --tab <id>` reuse the specified tab and make it the current tab for later unscoped operations.
- `--new-tab` and `newTab:true` force a new tab even if another tab is current.
- Commands that operate without `--tab` use the current tracked tab. Focusing or using a tab updates that current-tab pointer; if the pointer is stale, PinchTab falls back to the most recently used tracked tab.

Rationale: unscoped navigation should not accidentally replace the user or agent's active work surface. Reuse is explicit through `--tab`/`tabId`, while later unscoped read/action commands still have predictable current-tab behavior.

## Related Pages

- [Snapshot](./snapshot.md)
- [Tabs](./tabs.md)
