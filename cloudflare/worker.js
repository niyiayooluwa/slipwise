export default {
  async fetch(request, env, ctx) {
    if (request.method !== "GET") {
      return new Response("Method not allowed", { status: 405 });
    }

    const url = new URL(request.url);
    let targetUrl = "";

    if (url.pathname === "/live") {
      targetUrl = "https://www.sportybet.com/api/ng/factsCenter/configurableLiveOrPrematchEvents?sportId=sr:sport:1";
    } else if (url.pathname === "/ticket") {
      const code = url.searchParams.get("code");
      if (!code) return new Response("Missing code param", { status: 400 });
      targetUrl = `https://www.sportybet.com/api/ng/orders/share/${code}?_t=${Date.now()}`;
    } else {
      return new Response(JSON.stringify({
        pathname: url.pathname,
        search: url.search,
        href: url.href,
      }), { 
        status: 404,
        headers: { "Content-Type": "application/json" }
      });
    }

    // Check cache for live endpoint only
    const cache = caches.default;
    if (url.pathname === "/live") {
      const cached = await cache.match(targetUrl);
      if (cached) return cached;
    }

    try {
      const targetResponse = await fetch(targetUrl, {
        method: "GET",
        headers: {
          "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
          "Accept": "application/json",
          "Origin": "https://www.sportybet.com",
          "Referer": "https://www.sportybet.com/ng/",
        },
      });

      const body = await targetResponse.text();

      const response = new Response(body, {
        status: targetResponse.status,
        headers: {
          "Content-Type": "application/json",
          "Access-Control-Allow-Origin": "*",
          "Cache-Control": url.pathname === "/live" ? "max-age=60" : "no-store",
        },
      });

      if (url.pathname === "/live") {
        ctx.waitUntil(cache.put(targetUrl, response.clone()));
      }

      return response;

    } catch (error) {
      return new Response(JSON.stringify({ error: error.message }), { 
        status: 500,
        headers: { "Content-Type": "application/json" }
      });
    }
  },
};