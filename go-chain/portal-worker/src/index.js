const NODE_A = "https://tender-node.onrender.com";
const NODE_B = "https://tender-node-home.onrender.com";
const RELAY = "https://tender-relay.stephenwahogo0.workers.dev";

// TENDER design system — warm red/orange over near-black, geometric "T" monogram
const DASHBOARD = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>TENDER &mdash; Blockchain Network Portal</title>
<style>
  :root{
    --bg: #0b0d11;
    --bg-2: #101319;
    --surface: #171a21;
    --surface-2: #1e222b;
    --border: rgba(255,255,255,0.08);
    --border-hi: rgba(255,255,255,0.14);
    --text: #f3f5f8;
    --text-soft: #aab2c0;
    --text-dim: #6b7380;
    --red: #ff3b30;
    --orange: #ff7a18;
    --orange-soft: #ff9d3c;
    --grad: linear-gradient(100deg, #ff3b30 0%, #ff6b1a 45%, #ff9d3c 100%);
    --grad-glow: linear-gradient(100deg, rgba(255,59,48,.5), rgba(255,122,24,.5));
    --radius: 16px;
    --radius-sm: 10px;
    --shadow: 0 8px 32px rgba(0,0,0,.45);
    --font: "Inter","Segoe UI",system-ui,-apple-system,sans-serif;
    --mono: "JetBrains Mono","SF Mono",Consolas,monospace;
  }
  *{box-sizing:border-box;margin:0;padding:0}
  html{scroll-behavior:smooth}
  body{
    font-family:var(--font);
    background:
      radial-gradient(1200px 600px at 15% -10%, rgba(255,59,48,.08), transparent 60%),
      radial-gradient(1000px 500px at 90% -20%, rgba(255,122,24,.07), transparent 55%),
      var(--bg);
    color:var(--text);
    min-height:100vh;
    -webkit-font-smoothing:antialiased;
  }
  /* subtle grid texture */
  body::before{
    content:"";
    position:fixed;inset:0;pointer-events:none;z-index:0;
    background-image:linear-gradient(rgba(255,255,255,.02) 1px,transparent 1px),linear-gradient(90deg,rgba(255,255,255,.02) 1px,transparent 1px);
    background-size:44px 44px;
  }

  /* ---- Nav ---- */
  nav{
    position:sticky;top:0;z-index:50;
    background:rgba(11,13,17,.82);
    backdrop-filter:blur(14px);
    border-bottom:1px solid var(--border);
  }
  .nav-inner{
    max-width:1180px;margin:0 auto;padding:0 24px;height:64px;
    display:flex;align-items:center;justify-content:space-between;
  }
  .brand{display:flex;align-items:center;gap:12px;text-decoration:none;color:var(--text)}
  .brand-name{font-size:18px;font-weight:700;letter-spacing:.06em}
  .brand-name span{background:var(--grad);-webkit-background-clip:text;background-clip:text;-webkit-text-fill-color:transparent}
  .nav-links{display:flex;align-items:center;gap:8px}
  .nav-links a{
    color:var(--text-soft);text-decoration:none;font-size:13px;font-weight:500;
    padding:8px 14px;border-radius:8px;transition:all .15s;
  }
  .nav-links a:hover{color:var(--text);background:var(--surface)}
  .net-pill{
    display:flex;align-items:center;gap:8px;font-size:12px;font-weight:600;color:var(--text-soft);
    border:1px solid var(--border);padding:6px 12px;border-radius:999px;background:var(--surface);
  }
  .net-dot{width:8px;height:8px;border-radius:50%;background:#22c55e;box-shadow:0 0 10px rgba(34,197,94,.8);animation:pulse 2.4s infinite}
  @keyframes pulse{0%,100%{opacity:1}50%{opacity:.4}}

  /* ---- Hero ---- */
  header.hero{
    position:relative;z-index:1;
    max-width:1180px;margin:0 auto;padding:72px 24px 56px;
    text-align:center;
  }
  .hero h1{
    font-size:clamp(34px,6vw,58px);font-weight:800;letter-spacing:-.02em;line-height:1.06;
  }
  .hero h1 em{font-style:normal;background:var(--grad);-webkit-background-clip:text;background-clip:text;-webkit-text-fill-color:transparent}
  .hero p{
    margin:18px auto 0;max-width:640px;color:var(--text-soft);font-size:16px;line-height:1.6;
  }
  .hero-stats{
    margin-top:36px;display:flex;justify-content:center;gap:12px;flex-wrap:wrap;
  }
  .hero-chip{
    display:flex;align-items:center;gap:10px;padding:10px 18px;border-radius:12px;
    background:var(--surface);border:1px solid var(--border);font-size:13px;color:var(--text-soft);
  }
  .hero-chip b{color:var(--text);font-size:15px;font-weight:700}

  /* ---- Status grid ---- */
  main{position:relative;z-index:1;max-width:1180px;margin:0 auto;padding:0 24px 80px}
  .grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(330px,1fr));gap:20px}
  .card{
    background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);
    padding:24px;box-shadow:var(--shadow);transition:transform .18s,border-color .18s;
    position:relative;overflow:hidden;
  }
  .card::before{
    content:"";position:absolute;top:0;left:0;right:0;height:3px;
    background:var(--grad);opacity:0;transition:opacity .18s;
  }
  .card:hover{transform:translateY(-2px);border-color:var(--border-hi)}
  .card:hover::before{opacity:1}
  .card-head{display:flex;align-items:center;justify-content:space-between;margin-bottom:18px}
  .card-title{font-size:15px;font-weight:600;color:var(--text);display:flex;align-items:center;gap:10px}
  .node-tag{font-family:var(--mono);font-size:10px;color:var(--text-dim);border:1px solid var(--border);padding:3px 8px;border-radius:6px}
  .status{display:flex;align-items:center;gap:7px;font-size:12px;font-weight:600}
  .status .dot{width:9px;height:9px;border-radius:50%}
  .status.up{color:#4ade80}.status.up .dot{background:#22c55e;box-shadow:0 0 10px rgba(34,197,94,.7)}
  .status.down{color:#f87171}.status.down .dot{background:#ef4444;box-shadow:0 0 10px rgba(239,68,68,.7)}

  .stat{display:flex;justify-content:space-between;align-items:center;padding:11px 0;border-bottom:1px solid var(--border)}
  .stat:last-of-type{border-bottom:none}
  .stat .k{color:var(--text-soft);font-size:13px}
  .stat .v{color:var(--text);font-size:14px;font-weight:600;font-family:var(--mono)}
  .stat .v.acc{background:var(--grad);-webkit-background-clip:text;background-clip:text;-webkit-text-fill-color:transparent;font-weight:700}

  .card-actions{display:flex;gap:8px;margin-top:18px;flex-wrap:wrap}
  .btn{
    display:inline-flex;align-items:center;gap:6px;padding:9px 14px;border-radius:9px;
    font-size:13px;font-weight:600;text-decoration:none;transition:all .15s;
  }
  .btn.primary{background:var(--grad);color:#fff;box-shadow:0 4px 18px rgba(255,90,30,.35)}
  .btn.primary:hover{filter:brightness(1.08);transform:translateY(-1px)}
  .btn.ghost{color:var(--text-soft);border:1px solid var(--border);background:transparent}
  .btn.ghost:hover{color:var(--text);border-color:var(--border-hi);background:var(--surface-2)}

  .raw{
    margin-top:16px;background:#0b0d11;border:1px solid var(--border);border-radius:var(--radius-sm);
    padding:12px 14px;font-family:var(--mono);font-size:11.5px;color:var(--text-soft);
    white-space:pre;overflow-x:auto;max-height:220px;line-height:1.55;
  }

  footer{
    position:relative;z-index:1;border-top:1px solid var(--border);
    padding:28px 24px;text-align:center;color:var(--text-dim);font-size:13px;
  }
  footer a{color:var(--orange-soft);text-decoration:none}
  @media(max-width:640px){
    .nav-links a:not(.net-pill){display:none}
    .hero{padding-top:48px}
  }
</style>
</head>
<body>

<nav>
  <div class="nav-inner">
    <a class="brand" href="/">
      <svg width="34" height="34" viewBox="0 0 48 48" fill="none" aria-hidden="true">
        <defs>
          <linearGradient id="lg" x1="6" y1="6" x2="42" y2="42" gradientUnits="userSpaceOnUse">
            <stop offset="0" stop-color="#ff3b30"/>
            <stop offset=".5" stop-color="#ff6b1a"/>
            <stop offset="1" stop-color="#ff9d3c"/>
          </linearGradient>
        </defs>
        <path d="M24 2 L43 12 L43 22 L31 28 L31 40 L24 43 L17 40 L17 28 L5 22 L5 12 Z" stroke="url(#lg)" stroke-width="2.4" fill="rgba(255,90,30,.06)"/>
        <path d="M16 12 L32 12 M24 12 L24 34" stroke="url(#lg)" stroke-width="3.4" stroke-linecap="round"/>
        <circle cx="24" cy="11" r="1.6" fill="#fff"/>
      </svg>
      <span class="brand-name">TEN<span>DER</span></span>
    </a>
    <div class="nav-links">
      <a href="/a/">Node A</a>
      <a href="/b/">Node B</a>
      <a href="/r/peers">Relay</a>
      <span class="net-pill"><span class="net-dot"></span> tdr-mainnet-1</span>
    </div>
  </div>
</nav>

<header class="hero">
  <h1>One network.<br><em>Everything in view.</em></h1>
  <p>TENDER runs a pair of cloud-native validator nodes synced through a Cloudflare relay &mdash; no local device required. This single portal gives you live status for the whole network at a glance.</p>
  <div class="hero-stats">
    <span class="hero-chip">Network Height <b id="heroH">&#183;&#183;&#183;</b></span>
    <span class="hero-chip">Synced Nodes <b id="heroN">&#183;&#183;&#183;</b></span>
    <span class="hero-chip">Token Supply <b id="heroS">&#183;&#183;&#183;</b></span>
  </div>
</header>

<main>
  <div class="grid">

    <section class="card">
      <div class="card-head">
        <h2 class="card-title">Node A <span class="node-tag">tender-node</span></h2>
        <span class="status" id="stA"><span class="dot"></span>checking</span>
      </div>
      <div class="stat"><span class="k">Endpoint</span><span class="v">tender-node.onrender.com</span></div>
      <div class="stat"><span class="k">Block Height</span><span class="v acc" id="hA">&#183;&#183;&#183;</span></div>
      <div class="stat"><span class="k">Peers / Trusted</span><span class="v" id="pA">&#183;&#183;&#183;</span></div>
      <div class="stat"><span class="k">Token Supply</span><span class="v" id="sA">&#183;&#183;&#183;</span></div>
      <div class="stat"><span class="k">Consensus</span><span class="v" id="cA">&#183;&#183;&#183;</span></div>
      <div class="card-actions">
        <a class="btn primary" href="/a/">Open Node A</a>
        <a class="btn ghost" href="/a/api/chain">Chain JSON</a>
        <a class="btn ghost" href="/a/api/monitoring">Monitoring</a>
      </div>
    </section>

    <section class="card">
      <div class="card-head">
        <h2 class="card-title">Node B <span class="node-tag">tender-node-home</span></h2>
        <span class="status" id="stB"><span class="dot"></span>checking</span>
      </div>
      <div class="stat"><span class="k">Endpoint</span><span class="v">tender-node-home.onrender.com</span></div>
      <div class="stat"><span class="k">Block Height</span><span class="v acc" id="hB">&#183;&#183;&#183;</span></div>
      <div class="stat"><span class="k">Peers / Trusted</span><span class="v" id="pB">&#183;&#183;&#183;</span></div>
      <div class="stat"><span class="k">Token Supply</span><span class="v" id="sB">&#183;&#183;&#183;</span></div>
      <div class="stat"><span class="k">Consensus</span><span class="v" id="cB">&#183;&#183;&#183;</span></div>
      <div class="card-actions">
        <a class="btn primary" href="/b/">Open Node B</a>
        <a class="btn ghost" href="/b/api/chain">Chain JSON</a>
        <a class="btn ghost" href="/b/api/monitoring">Monitoring</a>
      </div>
    </section>

    <section class="card">
      <div class="card-head">
        <h2 class="card-title">P2P Relay <span class="node-tag">cloudflare worker</span></h2>
        <span class="status" id="stR"><span class="dot"></span>checking</span>
      </div>
      <div class="stat"><span class="k">Endpoint</span><span class="v">tender-relay.workers.dev</span></div>
      <div class="stat"><span class="k">Registered Nodes</span><span class="v" id="rN">&#183;&#183;&#183;</span></div>
      <div class="stat"><span class="k">Architecture</span><span class="v">Durable Object queue</span></div>
      <div class="raw" id="rRaw">&#8230;</div>
      <div class="card-actions">
        <a class="btn ghost" href="/r/peers">Relay /peers</a>
        <a class="btn ghost" href="/r/health">Health</a>
      </div>
    </section>

  </div>
</main>

<footer>
  TENDER &middot; cloud-native testnet &middot; two Render nodes, one Cloudflare relay &middot; <a href="https://tender-node.onrender.com">tender-node.onrender.com</a>
</footer>

<script>
async function get(url){try{const r=await fetch(url);return await r.json()}catch(e){return null}}
function setStatus(el,ok){const s=document.getElementById(el);s.className='status '+(ok?'up':'down');s.childNodes[0].className='dot';s.lastChild.textContent=ok?'online':'offline'}
function fmt(n){if(n===null||n===undefined)return '\u2013';return Number(n).toLocaleString('en-US')}
async function load(){
  const [a,b,r]=await Promise.all([
    get('/a/api/monitoring'),
    get('/b/api/monitoring'),
    get('/r/peers')
  ]);
  setStatus('stA',!!a);setStatus('stB',!!b);setStatus('stR',!!r);
  const put=(id,v)=>document.getElementById(id).textContent=v;
  if(a){put('hA',fmt(a.height));put('pA',(a.peer_count??0)+' / '+(a.trusted_peer_count??0));put('sA',fmt(a.token_supply));put('cA',(a.consensus||'pos').toUpperCase())}
  if(b){put('hB',fmt(b.height));put('pB',(b.peer_count??0)+' / '+(b.trusted_peer_count??0));put('sB',fmt(b.token_supply));put('cB',(b.consensus||'pos').toUpperCase())}
  if(r){const nodes=(r.nodes||[]);put('rN',nodes.length);put('rRaw',JSON.stringify(r,null,2))}
  const hA=a&&a.height||0,hB=b&&b.height||0;
  put('heroH',Math.max(hA,hB)||'\u2013');
  put('heroN',(a?1:0)+(b?1:0)+' / 2');
  put('heroS',fmt(a&&a.token_supply||b&&b.token_supply));
}
load();setInterval(load,8000);
</script>
</body>
</html>`;

export default {
	async fetch(request, env, ctx) {
		const url = new URL(request.url);
		const path = url.pathname;

		if (path === "/" || path === "") {
			return new Response(DASHBOARD, {
				status: 200,
				headers: { "Content-Type": "text/html; charset=utf-8" },
			});
		}
		if (path.startsWith("/r/")) {
			return proxy(RELAY + path.slice(2) + url.search, request);
		}
		if (path.startsWith("/a/")) {
			return proxy(NODE_A + path.slice(2) + url.search, request);
		}
		if (path.startsWith("/b/")) {
			return proxy(NODE_B + path.slice(2) + url.search, request);
		}
		return new Response("Not found", { status: 404 });
	},
};

async function proxy(target, request) {
	const headers = new Headers();
	for (const h of ["content-type", "accept", "authorization"]) {
		const v = request.headers.get(h);
		if (v) headers.set(h, v);
	}
	const init = { method: request.method, headers };
	if (!["GET", "HEAD"].includes(request.method)) {
		init.body = request.body;
	}
	const resp = await fetch(target, init);
	const newHeaders = new Headers(resp.headers);
	newHeaders.set("Access-Control-Allow-Origin", "*");
	return new Response(resp.body, { status: resp.status, headers: newHeaders });
}
