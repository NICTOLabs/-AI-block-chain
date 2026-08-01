import { RelayDO } from "./src/index.js";
import http from "http";

function makeStorage() {
	const map = new Map();
	return {
		async get(k) { return map.get(k); },
		async put(k, v) { map.set(k, v); },
	};
}

const storage = makeStorage();
const state = { storage };
const doObj = new RelayDO(state, {});

function toWorkerRequest(path, method, body) {
	return {
		url: "http://relay" + path,
		method,
		json: async () => (body ? JSON.parse(body) : {}),
	};
}

const server = http.createServer(async (req, res) => {
	const url = new URL(req.url, "http://relay");
	let body = "";
	req.on("data", (c) => (body += c));
	req.on("end", async () => {
		const id = url.searchParams.get("node") || "";
		let path = url.pathname;
		let reqBody = body || null;
		// Mirror the real worker's /push wrapper: inject the node id into the body.
		if (path === "/push") {
			const parsed = body ? JSON.parse(body) : {};
			reqBody = JSON.stringify({ node: id, msg: parsed });
		}
		const wreq = toWorkerRequest(path + url.search, req.method, reqBody);
		const resp = await doObj.fetch(wreq);
		const text = await resp.text();
		res.writeHead(resp.status, { "Content-Type": "application/json", "Access-Control-Allow-Origin": "*" });
		res.end(text);
	});
});

const port = parseInt(process.env.RELAY_PORT || "8788", 10);
server.listen(port, () => {
	console.log("mock relay listening on http://localhost:" + port);
});
