// Keeps the TikTok card's live action honest: polls /api/tiktok/live and flips
// the button between "Watch live" and a greyed-out "Offline" as the station's
// TikTok broadcast starts and ends, swapping the card's subtitle for the live
// room's own title while it runs.
//
// The server renders the correct state already, so this is only about the state
// changing under a page that's been open a while. It carries no policy: the
// endpoint's `live` is the display decision, unknown included (see
// tiktokLiveState in internal/handlers/public). Turbo re-executes this body
// <script> tag on every in-site navigation (see radio.js), so the setInterval
// loop is guarded the same way.
(function () {
  "use strict";

  var link = document.querySelector(".js-tiktok-live");
  if (!link) return;

  // Matches the server-side cache TTL in internal/tiktok: with the state cached
  // there, this is at most one upstream TikTok request per minute no matter how
  // many visitors are on the site.
  var POLL_MS = 60000;

  var dot = link.querySelector(".js-tiktok-dot");
  var label = link.querySelector(".js-tiktok-label");
  var tagline = document.querySelector(".js-tiktok-tagline");

  function apply(data) {
    link.classList.toggle("widget-action-live", data.live);
    link.classList.toggle("widget-action-off", !data.live);
    if (dot) dot.classList.toggle("motion-safe:animate-none", !data.live);
    if (label) label.textContent = data.live ? "Watch live" : "Offline";

    // An <a href> has no disabled state; pointer-events (via .widget-action-off)
    // stops the click, these two stop the keyboard and tell assistive tech.
    if (data.live) {
      link.removeAttribute("aria-disabled");
      link.removeAttribute("tabindex");
    } else {
      link.setAttribute("aria-disabled", "true");
      link.setAttribute("tabindex", "-1");
    }

    // title only ever arrives while genuinely live - TikTok keeps serving the
    // last room's title after a stream ends, so the server drops it off air.
    if (tagline) tagline.textContent = data.title || tagline.dataset.default;
  }

  function poll() {
    fetch("/api/tiktok/live", { headers: { "Accept": "application/json" } })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (data) { if (data) apply(data); })
      .catch(function () {});
  }

  poll();

  if (window.__tiktokLiveInit) return;
  window.__tiktokLiveInit = true;
  setInterval(poll, POLL_MS);
})();
