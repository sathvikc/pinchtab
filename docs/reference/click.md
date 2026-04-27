# Click

Click an element using a snapshot ref, CSS selector, XPath selector, text selector, or semantic selector.

```bash
curl -X POST http://localhost:9867/action \
  -H "Content-Type: application/json" \
  -d '{"kind":"click","ref":"e5"}'
# CLI Alternative
pinchtab click e5
# Response (use --json for full JSON)
OK
```

## CLI Flags

| Flag | Description |
|------|-------------|
| `--css` | CSS selector instead of ref |
| `--wait-nav` | Wait for navigation after click |
| `--snap` | Output interactive snapshot after click |
| `--snap-diff` | Output snapshot diff after click |
| `--text` | Output page text after click |
| `--dialog-action` | Auto-handle JS dialog: `accept` or `dismiss` |
| `--dialog-text` | Prompt response text (with `--dialog-action accept`) |
| `--x`, `--y` | Click at specific coordinates |
| `--json` | Full JSON response |
| `--tab` | Target specific tab |

## Examples

```bash
pinchtab click e5                       # Click by ref
pinchtab click "#login"                 # Click by CSS
pinchtab click "text:Submit"            # Click by text
pinchtab click e5 --snap                # Click and show new snapshot
pinchtab click e5 --wait-nav            # Click and wait for navigation
pinchtab click e5 --dialog-action accept  # Auto-accept alert/confirm
pinchtab click --x 100 --y 200          # Click at coordinates
```

## Notes

- Element refs come from `/snapshot`
- Refs for iframe descendants can be clicked directly without frame switch
- Selector lookup is limited to current frame scope (default: `main`)
- Use [`/frame`](./frame.md) before selector-based iframe actions
- Missing selectors fail immediately; use [`pinchtab wait`](./wait.md) first for dynamic UI
- The API also accepts `selector` field: `{"kind":"click","selector":"#login"}`
- Raw input is the default. To opt a click into the slower humanized path for a page that needs it, pass `humanize:true` in the action JSON or set `instanceDefaults.humanize:true`.

## Related Pages

- [Frame](./frame.md)
- [Snapshot](./snapshot.md)
- [Navigate](./navigate.md)
