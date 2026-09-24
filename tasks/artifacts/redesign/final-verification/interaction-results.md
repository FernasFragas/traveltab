# Interaction regression results

Tested 2026-09-24 against the working tree, default `cmd/design-preview` fixture, Brave `Chrome/153.0.8010.53`, Node 24.18.1, and the exact HTMX 1.9.11 file pinned by `views/index.go.tpl` (the script checks its SHA-384 integrity). The browser used an isolated temporary profile. External browser requests were blocked; iframe creation was verified, not remote playback.

The fixture listened on `127.0.0.1:18427` (app) and `127.0.0.1:18428` (map). The browser CDP port was `19437`. The passing command was:

```sh
HTMX_PATH=/private/tmp/traveltab-repair-htmx-1.9.11.js \
PREVIEW_URL=http://127.0.0.1:18427 CDP_PORT=19437 \
node tasks/redesign/checks/interaction-regressions.mjs
```

Result: exit 0. Valid video activation made one correctly titled `youtube-nocookie` iframe with the expected ID, including after delegated keyboard activation on Coimbra; invalid IDs made none and repeated activation made no duplicate. Porto showed its intended empty-video state, and the video list expanded and collapsed with matching ARIA state. Generate → Back → Forward → Back → generate preserved URL, city, form values, itinerary, stays, live status updates and focus. Two synchronous submissions made one request, and the later submission made a second. Restoring a real history entry without a planner caused no browser exception.

Negative controls used temporary copies of `HEAD` JavaScript served by the browser script's optional `PLANNER_PATH` and `DISCOVERY_PATH` overrides. The workspace JavaScript was not reverted:

| Variant | Command additions | Result |
| --- | --- | --- |
| Original planner and discovery | `PLANNER_PATH=/private/tmp/traveltab-original-planner-c.js DISCOVERY_PATH=/private/tmp/traveltab-original-discovery-c.js` | Exit 1 at initial video: iframe absent (`actual: undefined`). |
| Original planner, fixed discovery | `PLANNER_PATH=/private/tmp/traveltab-original-planner-c.js` | Initial video passed; exit 1 after Back: restored submit disabled, pending and busy true, stale “Planning your trip…” status. |
| Fixed planner and discovery | No overrides | Exit 0; all checks above passed. |

The original files were obtained with `git show HEAD:public/redesign/planner.js` and `git show HEAD:public/redesign/discovery.js`. Both negative controls used the same fixture, browser, HTMX file, and script as the passing run. `node --check tasks/redesign/checks/interaction-regressions.mjs` and `git diff --check` passed.
