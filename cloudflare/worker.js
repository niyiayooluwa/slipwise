/**
 * Simple Cloudflare Worker to proxy GET requests to bypass CORS/IP bans.
 */

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);

    // Only allow GET requests for the proxy
    if (request.method !== "GET") {
      return new Response("Method not allowed", { status: 405 });
    }

    // Determine the target URL based on the path or query parameters.
    // Example: /live could map to the SportyBet firehose endpoint.
    let targetUrl = "";
    
    if (url.pathname === "/live") {
      // Replace with actual SportyBet firehose URL
      targetUrl = "https://api.sportybet.com/api/v1/live/matches"; 
    } else if (url.searchParams.has("url")) {
      targetUrl = url.searchParams.get("url");
    } else {
      return new Response("Not found or invalid target", { status: 404 });
    }

    try {
      // Fetch data from the target URL
      const targetResponse = await fetch(targetUrl, {
        method: "GET",
        headers: {
          "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
          "Accept": "application/json",
          // Add other necessary headers for SportyBet
        },
      });

      // Forward the response back to the client
      const responseBody = await targetResponse.text();
      
      const headers = new Headers(targetResponse.headers);
      // Ensure CORS headers are set so the Go backend or frontend can access it if needed
      headers.set("Access-Control-Allow-Origin", "*");
      
      return new Response(responseBody, {
        status: targetResponse.status,
        headers: headers,
      });

    } catch (error) {
      return new Response(`Error fetching target: ${error.message}`, { status: 500 });
    }
  },
};
