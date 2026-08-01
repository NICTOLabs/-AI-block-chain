function makeStorage() {
	const map = new Map();
	return {
		async get(k) { return map.get(k); },
		async put(k, v) { map.set(k, v); },
		_dump() { return map; },
	};
}

function makeRequest(path, opts = {}) {
	const body = opts.body ? JSON.stringify(opts.body) : null;
	return {
		url: "http://relay" + path,
		json: async () => (body ? JSON.parse(body) : {}),
		method: opts.method || "GET",
	};
}

async function run() {
	const { RelayDO } = await import("./src/index.js");
	const storage = makeStorage();
	const state = { storage };
	const doObj = new RelayDO(state, {});

	// Node A pushes a block to broadcast
	await doObj.fetch(makeRequest("/push", { method: "POST", body: { node: "A", msg: { type: "block", block: { index: 1 }, to: "" } } }));
	// Node B pushes a tx
	await doObj.fetch(makeRequest("/push", { method: "POST", body: { node: "B", msg: { type: "tx", tx: { id: "x" } } } }));

	// Peers list should be A and B
	let peers = await doObj.fetch(makeRequest("/peers")).then((r) => r.json());
	console.log("peers:", JSON.stringify(peers));

	// A pulls: log-based relay returns both messages after A's cursor
	let a = await doObj.fetch(makeRequest("/pull?node=A")).then((r) => r.json());
	console.log("A received:", JSON.stringify(a));

	// B pulls: should get both (log after B cursor starts at 0)
	let b = await doObj.fetch(makeRequest("/pull?node=B")).then((r) => r.json());
	console.log("B received:", JSON.stringify(b));

	// Second pull should be empty (cursor advanced past all)
	let a2 = await doObj.fetch(makeRequest("/pull?node=A")).then((r) => r.json());
	console.log("A second pull (should be []):", JSON.stringify(a2));

	// Targeted message routes only to the target
	await doObj.fetch(makeRequest("/push", { method: "POST", body: { node: "A", msg: { type: "hello", to: "B" } } }));
	let b2 = await doObj.fetch(makeRequest("/pull?node=B")).then((r) => r.json());
	console.log("B targeted hello:", JSON.stringify(b2));
	let a3 = await doObj.fetch(makeRequest("/pull?node=A")).then((r) => r.json());
	console.log("A should NOT get targeted hello:", JSON.stringify(a3));

	// Late-joining node C gets full log history
	await doObj.fetch(makeRequest("/push", { method: "POST", body: { node: "C", msg: { type: "state", to: "" } } }));
	await doObj.fetch(makeRequest("/push", { method: "POST", body: { node: "D", msg: { type: "hello", to: "" } } }));
	let c = await doObj.fetch(makeRequest("/pull?node=C")).then((r) => r.json());
	console.log("C receives (full history):", JSON.stringify(c));
	let d = await doObj.fetch(makeRequest("/pull?node=D")).then((r) => r.json());
	console.log("D receives (full history):", JSON.stringify(d));

	const ok =
		peers.nodes.length === 2 &&
		a.length === 2 && a.some((m) => m.type === "tx") && a.some((m) => m.type === "block") &&
		b.length === 2 && b.some((m) => m.type === "tx") && b.some((m) => m.type === "block") &&
		a2.length === 0 &&
		b2.length === 1 && b2[0].type === "hello" &&
		a3.length === 0 &&
		c.length === 4 &&
		d.length === 4;
	console.log(ok ? "ALL TESTS PASSED" : "TESTS FAILED");
	process.exit(ok ? 0 : 1);
}

run().catch((e) => { console.error(e); process.exit(1); });
