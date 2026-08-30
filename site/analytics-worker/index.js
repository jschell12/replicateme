// Collect worker for the replicateme.cc marketing site.
// Runs on zone routes /a/e* in front of the Pages custom domain, so the
// beacon POST from site/script.js stays same-origin. Writes one row per
// page view or install-action event to the Workers Analytics Engine
// dataset "marketing_analytics".
//
// No personal data is stored. Raw IP and user-agent never leave this
// request handler; the only derived value kept is a SHA-256 hash salted
// with the current date (blob8), which exists solely to count unique
// visitors per day and cannot be linked across days.

const BOT_UA = /bot|crawl|spider|slurp|headless|lighthouse|pagespeed|preview|scan|python|curl|wget|httpx|monitor|uptime|facebookexternalhit/i;

export default {
  async fetch(request, env) {
    if (request.method !== "POST") return new Response(null, { status: 405 });
    try {
      const text = await request.text();
      if (text.length > 2048) return new Response(null, { status: 204 });
      const body = JSON.parse(text);
      const path = typeof body.p === "string" && body.p.startsWith("/") ? body.p.slice(0, 256) : null;
      const ua = request.headers.get("user-agent") || "";
      const purpose = request.headers.get("sec-purpose") || request.headers.get("purpose") || "";
      if (!path || !ua || BOT_UA.test(ua) || purpose.includes("prefetch")) return new Response(null, { status: 204 });
      let refHost = "";
      try {
        if (body.r) {
          const u = new URL(body.r);
          if (u.hostname !== new URL(request.url).hostname) refHost = u.hostname;
        }
      } catch {}
      const q = typeof body.q === "string" ? new URLSearchParams(body.q.slice(0, 512)) : new URLSearchParams();
      const event = typeof body.e === "string" && body.e.length <= 64 && /^[a-z0-9_]+$/.test(body.e) ? body.e : "";
      const ip = request.headers.get("cf-connecting-ip") || "";
      const day = new Date().toISOString().slice(0, 10);
      const salt = (env.ANALYTICS_SALT || "") + day;
      const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(`${salt}|replicateme|${ip}|${ua}`));
      const visitor = [...new Uint8Array(digest).slice(0, 8)].map((b) => b.toString(16).padStart(2, "0")).join("");
      // blob1 site, blob2 path, blob3 referrer host, blob4-6 utms, blob7 country,
      // blob8 daily visitor hash, blob9 event type ("" for pageviews, e.g. "install_copy"/"github_click").
      env.ANALYTICS.writeDataPoint({
        indexes: ["replicateme"],
        blobs: [
          "replicateme",
          path,
          refHost,
          q.get("utm_source") || "",
          q.get("utm_medium") || "",
          q.get("utm_campaign") || "",
          (request.cf && request.cf.country) || "",
          visitor,
          event,
        ],
      });
    } catch (e) {}
    return new Response(null, { status: 204 });
  },
};
