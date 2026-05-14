interface Env {
	ASSETS: { fetch(request: Request): Promise<Response> };
	ORIGIN: string;
}

const API_EXACT = new Set(['/mcp', '/health', '/metrics', '/docs']);

function isAPIPath(pathname: string): boolean {
	return pathname.startsWith('/api/') || pathname.startsWith('/docs/') || API_EXACT.has(pathname);
}

export default {
	async fetch(request: Request, env: Env): Promise<Response> {
		const url = new URL(request.url);
		const { pathname, search } = url;

		// 1. Try static assets first (landing page, CSS, JS, images)
		const assetResponse = await env.ASSETS.fetch(request);
		if (assetResponse.status !== 404) return assetResponse;

		// 2. Proxy to Go origin (ORIGIN var: Fly.io in prod, localhost:8080 in dev)
		console.info(`Proxying request to origin: ${pathname}${search} with env ${env.ORIGIN}`);
		const originResponse = await fetch(`${env.ORIGIN}${pathname}${search}`, {
			method: request.method,
			headers: request.headers,
			body: request.body,
			redirect: 'manual',
		});

		// 3. API/server paths: return Go's response as-is (JSON errors expected by clients)
		if (isAPIPath(pathname)) return originResponse;

		// 4. Short-code paths: 3xx/2xx pass through; 404/410 → Astro 404 page
		if (originResponse.status === 404 || originResponse.status === 410) {
			const notFoundPage = await env.ASSETS.fetch(new Request(`${url.origin}/404`));
			return new Response(notFoundPage.body, {
				status: originResponse.status,
				headers: notFoundPage.headers,
			});
		}

		return originResponse;
	},
};
