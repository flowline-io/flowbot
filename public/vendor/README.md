# Vendored frontend assets

Pinned copies served from `/static/vendor/*`. Refresh most files with `scripts/vendor.sh`.

## ECharts custom build

| File | Version | Contents | Approx size |
| --- | --- | --- | --- |
| `echarts.custom.min.js` | 5.6.0 | `graph`, `sankey`, `tree`, `tooltip`, `legend`, `dataZoom`, `CanvasRenderer` | ~577 KiB |

Rebuild (from a scratch directory):

```bash
npm install echarts@5.6.0 esbuild@0.25.0
# entry.js imports the charts/components above and assigns window.echarts
npx esbuild entry.js --bundle --minify --outfile=echarts.custom.min.js --format=iife
```

Copy the outfile to this directory. Init lives in `public/js/flow-graph.js` (no inline page scripts).
