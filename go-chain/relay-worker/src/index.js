export default {
	async fetch(request, env, ctx) {
		const url = new URL(request.url);
		const path = url.pathname;
		const id = url.searchParams.get("node") || "";
		const origin = request.headers.get("Origin") || "";
		const headers = {
			"Access-Control-Allow-Origin": origin,
			"Access-Control-Allow-Methods": "GET, POST, OPTIONS",
			"Access-Control-Allow-Headers": "Content-Type, Authorization",
		};

		if (request.method === "OPTIONS") {
			return new Response(null, { status: 204, headers });
		}

		try {
			if (path === "/push" && request.method === "POST") {
				if (!id) return new Response(JSON.stringify({ error: "missing node" }), { status: 400, headers });
				const body = await request.json();
				const stub = env.RELAY.get(env.RELAY.idFromName("relay"));
				await stub.fetch("http://relay/push", {
					method: "POST",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({ node: id, msg: body }),
				});
				return new Response(JSON.stringify({ ok: true }), { status: 200, headers });
			}

			if (path === "/pull" && request.method === "GET") {
				if (!id) return new Response(JSON.stringify({ error: "missing node" }), { status: 400, headers });
				const stub = env.RELAY.get(env.RELAY.idFromName("relay"));
				const resp = await stub.fetch(`http://relay/pull?node=${encodeURIComponent(id)}`);
				const msgs = await resp.json();
				return new Response(JSON.stringify(msgs), { status: 200, headers });
			}

			if (path === "/peers" && request.method === "GET") {
				const stub = env.RELAY.get(env.RELAY.idFromName("relay"));
				const resp = await stub.fetch("http://relay/peers");
				const peers = await resp.json();
				return new Response(JSON.stringify(peers), { status: 200, headers });
			}

			if (path === "/health") {
				return new Response(JSON.stringify({ status: "ok" }), { status: 200, headers });
			}

			return new Response(JSON.stringify({ error: "not found" }), { status: 404, headers });
		} catch (err) {
			return new Response(JSON.stringify({ error: String(err) }), { status: 500, headers });
		}
	},
};

export class RelayDO {
	constructor(state, env) {
		this.state = state;
		this.storage = state.storage;
	}

	async fetch(request) {
		const url = new URL(request.url);
		const path = url.pathname;

		if (path === "/push") {
			const { node, msg } = await request.json();
			let nodes = (await this.storage.get("nodes")) || [];
			if (!nodes.includes(node)) {
				nodes = [...nodes, node];
				await this.storage.put("nodes", nodes);
			}
			const target = (msg && msg.to) || null;
			if (target) {
				const queue = (await this.storage.get("q:" + target)) || [];
				queue.push({ node, msg, ts: Date.now() });
				if (queue.length > 5000) queue.splice(0, queue.length - 5000);
				await this.storage.put("q:" + target, queue);
				return new Response(JSON.stringify({ ok: true }), { status: 200 });
			}
			let seq = (await this.storage.get("lastSeq")) || 0;
			seq += 1;
			let log = (await this.storage.get("log")) || [];
			log.push({ seq, node, msg, ts: Date.now() });
			if (log.length > 5000) log.splice(0, log.length - 5000);
			await this.storage.put("log", log);
			await this.storage.put("lastSeq", seq);
			return new Response(JSON.stringify({ ok: true }), { status: 200 });
		}

		if (path === "/pull") {
			const node = url.searchParams.get("node") || "";
			const log = (await this.storage.get("log")) || [];
			const cursor = (await this.storage.get("cursor:" + node)) || 0;
			let out = log.filter((e) => e.seq > cursor).map((e) => e.msg);
			const queue = (await this.storage.get("q:" + node)) || [];
			if (queue.length > 0) {
				out = out.concat(queue.map((e) => e.msg));
				await this.storage.put("q:" + node, []);
			}
			let last = 0;
			if (log.length > 0) last = log[log.length - 1].seq;
			await this.storage.put("cursor:" + node, last);
			return new Response(JSON.stringify(out), { status: 200 });
		}

		if (path === "/peers") {
			const nodes = (await this.storage.get("nodes")) || [];
			return new Response(JSON.stringify({ nodes }), { status: 200 });
		}

		return new Response(JSON.stringify({ error: "not found" }), { status: 404 });
	}
}
