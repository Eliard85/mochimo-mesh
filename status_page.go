package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

type statusPageData struct {
	APIBase               string
	Blockchain            string
	Network               string
	NodeVersion           string
	MiddlewareVersion     string
	OnlineMode            bool
	Synced                bool
	SyncStage             string
	LatestBlockNum        uint64
	LatestBlockHash       string
	CurrentBlockUnixMilli uint64
	CurrentBlockAge       string
	CurrentDifficulty     uint32
	SuggestedFee          uint64
	HTTPPort              int
	HTTPSPort             int
	EnableHTTPS           bool
	LastUpdated           string
	RecentBlocks          []RecentBlockSummary
}

var dashboardTimeZone = time.FixedZone("GMT+3", 3*60*60)

var statusPageTemplate = template.Must(template.New("status-page").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Mochimo Mesh Node Status</title>
  <style>
    :root {
      --paper: #f6f1e7;
      --paper-strong: #fffaf2;
      --ink: #1f2a2a;
      --muted: #5e6b67;
      --panel: rgba(255, 250, 242, 0.86);
      --panel-border: rgba(31, 42, 42, 0.12);
      --accent: #0d7a6f;
      --accent-soft: rgba(13, 122, 111, 0.12);
      --warn: #b4582e;
      --warn-soft: rgba(180, 88, 46, 0.12);
      --good: #2f7f45;
      --good-soft: rgba(47, 127, 69, 0.12);
      --shadow: 0 18px 50px rgba(31, 42, 42, 0.10);
      --radius: 22px;
    }
    * {
      box-sizing: border-box;
    }
    body {
      margin: 0;
      min-height: 100vh;
      font-family: "Trebuchet MS", "Lucida Sans Unicode", sans-serif;
      color: var(--ink);
      background:
        radial-gradient(circle at top left, rgba(13, 122, 111, 0.18), transparent 30%),
        radial-gradient(circle at top right, rgba(180, 88, 46, 0.16), transparent 24%),
        linear-gradient(135deg, #efe6d4 0%, #f8f3ea 48%, #ebe2cf 100%);
    }
    .shell {
      max-width: 1180px;
      margin: 0 auto;
      padding: 28px 18px 48px;
    }
    .hero {
      position: relative;
      overflow: hidden;
      padding: 30px;
      border: 1px solid var(--panel-border);
      border-radius: calc(var(--radius) + 6px);
      background: linear-gradient(150deg, rgba(255, 250, 242, 0.94), rgba(247, 238, 224, 0.82));
      box-shadow: var(--shadow);
    }
    .hero::after {
      content: "";
      position: absolute;
      inset: auto -8% -45% auto;
      width: 340px;
      height: 340px;
      border-radius: 999px;
      background: radial-gradient(circle, rgba(13, 122, 111, 0.14), transparent 68%);
      pointer-events: none;
    }
    .eyebrow {
      font-size: 0.84rem;
      letter-spacing: 0.16em;
      text-transform: uppercase;
      color: var(--muted);
      margin-bottom: 10px;
    }
    h1 {
      margin: 0 0 10px;
      font-family: Georgia, "Times New Roman", serif;
      font-size: clamp(2rem, 4vw, 3.8rem);
      line-height: 0.98;
    }
    .hero p {
      max-width: 720px;
      margin: 0;
      color: var(--muted);
      font-size: 1rem;
      line-height: 1.6;
    }
    .status-strip {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-top: 22px;
    }
    .pill {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 10px 14px;
      border-radius: 999px;
      border: 1px solid transparent;
      background: var(--accent-soft);
      font-size: 0.94rem;
      white-space: nowrap;
    }
    .pill.good {
      background: var(--good-soft);
      color: #205734;
    }
    .pill.warn {
      background: var(--warn-soft);
      color: #7e3f20;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(12, minmax(0, 1fr));
      gap: 16px;
      margin-top: 18px;
    }
    .card {
      grid-column: span 12;
      padding: 20px;
      border-radius: var(--radius);
      border: 1px solid var(--panel-border);
      background: var(--panel);
      box-shadow: var(--shadow);
      backdrop-filter: blur(10px);
      animation: rise 380ms ease both;
    }
    .card:nth-child(2) { animation-delay: 70ms; }
    .card:nth-child(3) { animation-delay: 120ms; }
    .card:nth-child(4) { animation-delay: 170ms; }
    .card:nth-child(5) { animation-delay: 220ms; }
    @keyframes rise {
      from {
        opacity: 0;
        transform: translateY(14px);
      }
      to {
        opacity: 1;
        transform: translateY(0);
      }
    }
    .card h2 {
      margin: 0 0 16px;
      font-family: Georgia, "Times New Roman", serif;
      font-size: 1.25rem;
    }
    .metrics {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 14px;
    }
    .metrics-four {
      grid-template-columns: repeat(5, minmax(0, 1fr));
    }
    .metrics-live {
      grid-template-columns: max-content max-content max-content minmax(0, 1fr) max-content;
    }
    .metric {
      padding: 14px;
      border-radius: 16px;
      background: rgba(255, 255, 255, 0.52);
      border: 1px solid rgba(31, 42, 42, 0.08);
    }
    .metric-compact .label,
    .metric-compact .value {
      white-space: nowrap;
    }
    .label {
      display: block;
      margin-bottom: 8px;
      color: var(--muted);
      font-size: 0.78rem;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }
    .value {
      font-size: 1.08rem;
      line-height: 1.45;
      word-break: break-word;
    }
    .mono {
      font-family: "Lucida Console", "Courier New", monospace;
      font-size: 0.95rem;
    }
    .wide {
      grid-column: span 12;
    }
    .actions {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-top: 8px;
    }
    .button {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      padding: 11px 15px;
      border-radius: 14px;
      border: 1px solid rgba(31, 42, 42, 0.12);
      background: rgba(255, 255, 255, 0.76);
      color: var(--ink);
      text-decoration: none;
      cursor: pointer;
    }
    .button:hover {
      background: rgba(13, 122, 111, 0.08);
    }
    pre {
      margin: 14px 0 0;
      padding: 14px;
      overflow-x: auto;
      border-radius: 16px;
      background: #1f2626;
      color: #f6f1e7;
      font-family: "Lucida Console", "Courier New", monospace;
      font-size: 0.89rem;
      line-height: 1.5;
    }
    .footer-note {
      margin-top: 18px;
      color: var(--muted);
      font-size: 0.92rem;
    }
    .hidden {
      display: none;
    }
    .table-shell {
      overflow-x: hidden;
      border-radius: 18px;
      border: 1px solid rgba(31, 42, 42, 0.08);
      background: rgba(255, 255, 255, 0.48);
    }
    table {
      width: 100%;
      border-collapse: collapse;
      table-layout: fixed;
    }
    .recent-blocks-table col.col-height {
      width: 9ch;
    }
    .recent-blocks-table col.col-solved {
      width: 190px;
    }
    .recent-blocks-table col.col-mined {
      width: 32%;
    }
    .recent-blocks-table col.col-hash {
      width: auto;
    }
    th,
    td {
      padding: 13px 14px;
      text-align: left;
      vertical-align: top;
      border-bottom: 1px solid rgba(31, 42, 42, 0.08);
    }
    th {
      color: var(--muted);
      font-size: 0.76rem;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }
    tbody tr:last-child td {
      border-bottom: none;
    }
    .primary {
      font-weight: 600;
    }
    .subtle {
      margin-top: 4px;
      color: var(--muted);
      font-size: 0.85rem;
      line-height: 1.45;
    }
    .wrap-cell {
      white-space: normal;
      overflow-wrap: anywhere;
      word-break: break-all;
    }
    .nowrap-cell {
      white-space: nowrap;
    }
    @media (max-width: 1080px) {
      .metrics-four,
      .metrics-live {
        grid-template-columns: repeat(2, minmax(0, 1fr));
      }
    }
    @media (max-width: 860px) {
      .metrics {
        grid-template-columns: 1fr;
      }
      .hero {
        padding: 24px;
      }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <div class="eyebrow">Mochimo Mesh Browser View</div>
      <h1>Node status at a glance</h1>
      <p>This page reads the same Mesh API that you query with <code>curl</code>. It auto-refreshes every 5 seconds and is meant for quick checks from a browser.</p>
      <div class="status-strip">
        <span id="pill-mode" class="pill {{if .OnlineMode}}good{{else}}warn{{end}}">Mode: {{if .OnlineMode}}online{{else}}offline{{end}}</span>
        <span id="pill-sync" class="pill {{if .Synced}}good{{else}}warn{{end}}">Sync: {{if .Synced}}yes{{else}}no{{end}}</span>
        <span class="pill">Network: {{.Blockchain}} / {{.Network}}</span>
        <span class="pill">HTTP: {{.HTTPPort}}</span>
        <span id="pill-https" class="pill {{if .EnableHTTPS}}good{{else}}warn{{end}}">HTTPS: {{if .EnableHTTPS}}on{{else}}off{{end}}</span>
      </div>
    </section>

    <section class="grid">
      <article class="card wide">
        <h2>Live chain data</h2>
        <div class="metrics metrics-four metrics-live">
          <div class="metric metric-compact">
            <span class="label">Current Block</span>
            <div id="latest-block" class="value">{{.LatestBlockNum}}</div>
          </div>
          <div class="metric metric-compact">
            <span class="label">Block Age</span>
            <div id="block-age" class="value">{{.CurrentBlockAge}}</div>
          </div>
          <div class="metric metric-compact">
            <span class="label">Difficulty</span>
            <div id="current-difficulty" class="value">{{.CurrentDifficulty}}</div>
          </div>
          <div class="metric">
            <span class="label">Current Block Hash</span>
            <div id="latest-hash" class="value mono">{{.LatestBlockHash}}</div>
          </div>
          <div class="metric metric-compact">
            <span class="label">Suggested Fee</span>
            <div class="value">{{.SuggestedFee}}</div>
          </div>
        </div>
      </article>

      <article class="card wide">
        <h2>Mesh state</h2>
        <div class="metrics">
          <div class="metric">
            <span class="label">Sync Stage</span>
            <div id="sync-stage" class="value">{{.SyncStage}}</div>
          </div>
          <div class="metric">
            <span class="label">Last Refresh</span>
            <div id="last-updated" class="value">{{.LastUpdated}}</div>
          </div>
        </div>
      </article>

      <article class="card wide">
        <h2>Latest 50 blocks on this node</h2>
        <div class="table-shell">
          <table class="recent-blocks-table">
            <colgroup>
              <col class="col-height">
              <col class="col-solved">
              <col class="col-mined">
              <col class="col-hash">
            </colgroup>
            <thead>
              <tr>
                <th>Height</th>
                <th>Solved</th>
                <th>Mined To</th>
                <th>Hash</th>
              </tr>
            </thead>
            <tbody id="recent-blocks-body">
              {{range .RecentBlocks}}
              <tr>
                <td class="primary nowrap-cell">{{.Number}}</td>
                <td>
                  <div class="primary">{{.TimestampLabel}}</div>
                  <div class="subtle">{{.Age}}</div>
                </td>
                <td class="wrap-cell">
                  <div class="primary mono">{{if .MinerAddressBase58}}{{.MinerAddressBase58}}{{else}}{{.MinerAddress}}{{end}}</div>
                  {{if .MinerAddressBase58}}<div class="subtle mono">{{.MinerAddress}}</div>{{end}}
                </td>
                <td class="mono wrap-cell">{{.Hash}}</td>
              </tr>
              {{end}}
            </tbody>
          </table>
        </div>
        <div id="recent-blocks-empty" class="footer-note {{if .RecentBlocks}}hidden{{end}}">No recent block data is available yet.</div>
      </article>

      <article class="card wide">
        <h2>Runtime configuration</h2>
        <div class="metrics">
          <div class="metric">
            <span class="label">API Base</span>
            <div class="value mono">{{.APIBase}}</div>
          </div>
          <div class="metric">
            <span class="label">Versions</span>
            <div class="value">Node {{.NodeVersion}} · Mesh {{.MiddlewareVersion}}</div>
          </div>
        </div>
      </article>

      <article class="card wide">
        <h2>Quick actions</h2>
        <div class="actions">
          <button id="refresh-btn" class="button" type="button">Refresh now</button>
          <button id="network-list-btn" class="button" type="button">Show /network/list</button>
          <button id="current-block-btn" class="button" type="button">Show current block</button>
          <a class="button" href="/dashboard" target="_self">Reload dashboard</a>
        </div>
        <pre id="status-json">Use the buttons above to inspect JSON payloads.</pre>
        <div class="footer-note">The dashboard refreshes itself through <code>GET /dashboard/data</code>. Buttons below still query the Mesh API endpoints directly.</div>
      </article>
    </section>
  </div>

  <script>
    const state = {
      currentBlockTimestamp: {{.CurrentBlockUnixMilli}},
      onlineMode: {{if .OnlineMode}}true{{else}}false{{end}},
    };

    function formatAge(ts) {
      if (!ts) return "unknown";
      const diffSeconds = Math.max(0, Math.floor((Date.now() - ts) / 1000));
      if (diffSeconds < 60) return diffSeconds + " sec ago";
      if (diffSeconds < 3600) return Math.floor(diffSeconds / 60) + " min ago";
      if (diffSeconds < 86400) return Math.floor(diffSeconds / 3600) + " h ago";
      return Math.floor(diffSeconds / 86400) + " d ago";
    }

	    function updateAge() {
	      document.getElementById("block-age").textContent = formatAge(state.currentBlockTimestamp);
	      document.querySelectorAll("[data-block-age]").forEach((node) => {
	        const ts = Number(node.getAttribute("data-ts") || "0");
	        node.textContent = formatAge(ts);
	      });
	    }

    function setBadge(id, ok, goodText, badText) {
      const node = document.getElementById(id);
      node.classList.toggle("good", ok);
      node.classList.toggle("warn", !ok);
      node.textContent = ok ? goodText : badText;
    }

    async function postJSON(path, payload) {
      const response = await fetch(path, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
      });
      const text = await response.text();
      document.getElementById("status-json").textContent = text;
      if (!response.ok) {
        throw new Error("HTTP " + response.status);
      }
      return text;
    }

    async function fetchJSON(path, mirrorToStatus = false) {
      const response = await fetch(path, {
        method: "GET",
        headers: { "Accept": "application/json" }
      });
      const text = await response.text();
      if (mirrorToStatus) {
        document.getElementById("status-json").textContent = text;
      }
      if (!response.ok) {
        throw new Error("HTTP " + response.status);
      }
      return JSON.parse(text);
    }

    function escapeHTML(value) {
      return String(value ?? "").replace(/[&<>"']/g, (char) => ({
        "&": "&amp;",
        "<": "&lt;",
        ">": "&gt;",
        "\"": "&quot;",
        "'": "&#39;"
      }[char]));
    }

	function renderRecentBlocks(blocks) {
	      const body = document.getElementById("recent-blocks-body");
	      const empty = document.getElementById("recent-blocks-empty");

	      if (!Array.isArray(blocks) || blocks.length === 0) {
        body.innerHTML = "";
        empty.classList.remove("hidden");
        return;
	      }

	      empty.classList.add("hidden");
	      const rows = blocks.map((block) => {
	        const number = block.Number ?? block.number;
	        const hash = block.Hash ?? block.hash;
	        const timestampLabel = block.TimestampLabel ?? block.timestamp_label;
	        const age = block.Age ?? block.age;
	        const timestampUnixMilli = block.TimestampUnixMilli ?? block.timestamp_unix_milli ?? 0;
	        const minerAddress = block.MinerAddress ?? block.miner_address;
	        const minerAddressBase58 = block.MinerAddressBase58 ?? block.miner_address_base58;

	        if (!hash || /^0x0+$/i.test(hash)) {
	          return "";
	        }

	        const minedTo = minerAddressBase58 || minerAddress || "n/a";
	        const minedToHex = minerAddressBase58
	          ? '<div class="subtle mono">' + escapeHTML(minerAddress || "") + '</div>'
	          : "";

	        const ageLabel = timestampUnixMilli ? formatAge(timestampUnixMilli) : (age || "unknown");

	        return ''
	          + '<tr>'
	          +   '<td class="primary nowrap-cell">' + escapeHTML(number ?? "n/a") + '</td>'
	          +   '<td>'
	          +     '<div class="primary">' + escapeHTML(timestampLabel || "unknown") + '</div>'
	          +     '<div class="subtle" data-block-age="1" data-ts="' + escapeHTML(timestampUnixMilli) + '">' + escapeHTML(ageLabel) + '</div>'
	          +   '</td>'
	          +   '<td class="wrap-cell">'
	          +     '<div class="primary mono">' + escapeHTML(minedTo) + '</div>'
	          +     minedToHex
	          +   '</td>'
	          +   '<td class="mono wrap-cell">' + escapeHTML(hash) + '</td>'
	          + '</tr>';
	      }).filter(Boolean);

	      if (rows.length === 0) {
	        body.innerHTML = "";
	        empty.classList.remove("hidden");
	        return;
	      }

	      body.innerHTML = rows.join("");
	    }

    async function refreshStatus(mirrorToStatus = false) {
      const data = await fetchJSON("/dashboard/data", mirrorToStatus);
      state.currentBlockTimestamp = data.CurrentBlockUnixMilli || 0;
      state.onlineMode = Boolean(data.OnlineMode);

      document.getElementById("latest-block").textContent = data.LatestBlockNum ?? "n/a";
      document.getElementById("latest-hash").textContent = data.LatestBlockHash ?? "n/a";
      document.getElementById("current-difficulty").textContent = data.CurrentDifficulty ?? "n/a";
      document.getElementById("sync-stage").textContent = data.SyncStage ?? "unknown";
      document.getElementById("last-updated").textContent = data.LastUpdated ?? "unknown";

      setBadge("pill-mode", Boolean(data.OnlineMode), "Mode: online", "Mode: offline");
      setBadge("pill-sync", Boolean(data.Synced), "Sync: yes", "Sync: no");
      setBadge("pill-https", Boolean(data.EnableHTTPS), "HTTPS: on", "HTTPS: off");
      renderRecentBlocks(data.RecentBlocks || []);

      updateAge();
    }

    document.getElementById("refresh-btn").addEventListener("click", () => {
      refreshStatus(true).catch((err) => {
        document.getElementById("status-json").textContent = "Refresh failed: " + err.message;
      });
    });

    document.getElementById("network-list-btn").addEventListener("click", () => {
      postJSON("/network/list", {}).catch((err) => {
        document.getElementById("status-json").textContent = "Request failed: " + err.message;
      });
    });

    document.getElementById("current-block-btn").addEventListener("click", () => {
      if (!state.onlineMode) {
        document.getElementById("status-json").textContent = "Mesh runs in offline mode. Current block query is unavailable.";
        return;
      }

      postJSON("/block", {
        network_identifier: {
          blockchain: "{{.Blockchain}}",
          network: "{{.Network}}"
        },
        block_identifier: {
          index: 0,
          hash: ""
        }
      }).catch((err) => {
        document.getElementById("status-json").textContent = "Request failed: " + err.message;
      });
    });

    updateAge();
    renderRecentBlocks({{if .RecentBlocks}}[
      {{- range $index, $block := .RecentBlocks }}
      {{- if $index }},{{ end }}
      {
        "Number": {{ $block.Number }},
        "TimestampUnixMilli": {{ $block.TimestampUnixMilli }},
        "TimestampLabel": {{ printf "%q" $block.TimestampLabel }},
        "Age": {{ printf "%q" $block.Age }},
        "Hash": {{ printf "%q" $block.Hash }},
        "MinerAddress": {{ printf "%q" $block.MinerAddress }},
        "MinerAddressBase58": {{ printf "%q" $block.MinerAddressBase58 }}
      }
      {{- end }}
    ]{{else}}[]{{end}});
    refreshStatus(false).catch((err) => {
      document.getElementById("status-json").textContent = "Refresh failed: " + err.message;
    });
    setInterval(updateAge, 1000);
    setInterval(() => {
      refreshStatus(false).catch((err) => {
        document.getElementById("status-json").textContent = "Refresh failed: " + err.message;
      });
    }, 5000);
  </script>
</body>
</html>
`))

func statusPageHandler(w http.ResponseWriter, r *http.Request) {
	data := buildStatusPageData(r)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := statusPageTemplate.Execute(w, data); err != nil {
		mlog(3, "§bstatusPageHandler(): §4Error rendering status page: §c%s", err)
		http.Error(w, "failed to render status page", http.StatusInternalServerError)
	}
}

func dashboardDataHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(buildStatusPageData(r)); err != nil {
		mlog(3, "§bdashboardDataHandler(): §4Error encoding dashboard data: §c%s", err)
		http.Error(w, "failed to encode dashboard data", http.StatusInternalServerError)
	}
}

func buildStatusPageData(r *http.Request) statusPageData {
	apiBase := fmt.Sprintf("%s://%s", requestScheme(r), r.Host)

	return statusPageData{
		APIBase:               apiBase,
		Blockchain:            Constants.NetworkIdentifier.Blockchain,
		Network:               Constants.NetworkIdentifier.Network,
		NodeVersion:           Constants.NetworkOptionsResponseVersion.NodeVersion,
		MiddlewareVersion:     Constants.NetworkOptionsResponseVersion.MiddlewareVersion,
		OnlineMode:            Globals.OnlineMode,
		Synced:                Globals.IsSynced,
		SyncStage:             Globals.LastSyncStage,
		LatestBlockNum:        Globals.LatestBlockNum,
		LatestBlockHash:       "0x" + hex.EncodeToString(Globals.LatestBlockHash[:]),
		CurrentBlockUnixMilli: Globals.CurrentBlockUnixMilli,
		CurrentBlockAge:       formatBlockAge(Globals.CurrentBlockUnixMilli),
		CurrentDifficulty:     Globals.CurrentDifficulty,
		SuggestedFee:          Globals.SuggestedFee,
		HTTPPort:              Globals.HTTPPort,
		HTTPSPort:             Globals.HTTPSPort,
		EnableHTTPS:           Globals.EnableHTTPS,
		LastUpdated:           formatDashboardTime(time.Now()),
		RecentBlocks:          getRecentBlocksSnapshot(),
	}
}

func requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func formatBlockAge(ts uint64) string {
	if ts == 0 {
		return "unknown"
	}

	diff := time.Since(time.UnixMilli(int64(ts)))
	if diff < 0 {
		diff = 0
	}
	if diff < time.Minute {
		return fmt.Sprintf("%d sec ago", int(diff.Seconds()))
	}
	if diff < time.Hour {
		return fmt.Sprintf("%d min ago", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("%d h ago", int(diff.Hours()))
	}
	return fmt.Sprintf("%d d ago", int(diff.Hours()/24))
}

func formatDashboardTime(t time.Time) string {
	return t.In(dashboardTimeZone).Format("2006-01-02 15:04:05 MST")
}
