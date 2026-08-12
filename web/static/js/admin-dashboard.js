// Keeps the admin dashboard's on-air strip live: polls the same public endpoints
// the site itself uses (/api/schedule/current for the program, /api/nowplaying for
// the track and stream state) and updates the card in place. Anything the public
// site already exposes is read from the public endpoint on purpose - duplicating it
// behind /admin would mean two things to keep in step.
//
// The one exception is /admin/api/listeners: the audience count is deliberately
// kept off the public API, so there is no public endpoint to read it from.
//
// Loaded on every admin page (the layout has one script list), so it exits
// immediately when the dashboard's card isn't on the page.
(function () {
  "use strict";

  var card = document.querySelector(".js-dash-onair");
  if (!card) return;

  var POLL_MS = 30000;

  function el(sel) { return card.querySelector(sel); }

  function show(node, visible) {
    if (node) node.classList.toggle("hidden", !visible);
  }

  function text(sel, value) {
    var node = el(sel);
    if (node) node.textContent = value;
  }

  function applySchedule(data) {
    show(el(".js-onair-badge"), !!data.on_air);
    show(el(".js-onair-idle"), !data.on_air);
    show(el(".js-onair-card"), !!data.on_air);
    show(el(".js-onair-empty"), !data.on_air);
    if (!data.on_air) return;

    text(".js-schedule-title", data.title || "");
    text(".js-schedule-time", (data.start || "") + "–" + (data.end || ""));
    text(".js-schedule-host", data.host ? "· " + data.host : "");

    var img = el(".js-schedule-img");
    if (img && data.image && img.getAttribute("src") !== data.image) {
      img.src = data.image;
    }
    show(el(".js-schedule-thumb"), true);
    show(img, !!data.image);
    show(el(".js-schedule-icon"), !data.image);

    // Width is a safelisted w-[N%] utility, never an inline style: the CSP has no
    // style-src 'unsafe-inline'. Same swap the public card does.
    var bar = el(".js-progress-bar");
    if (bar) bar.className = bar.className.replace(/\bw-\[\d+%\]/, "w-[" + (data.progress || 0) + "%]");
  }

  function applyNowPlaying(data) {
    var badge = el(".js-stream-badge");
    if (badge) {
      badge.classList.toggle("bg-white/15", !!data.live);
      badge.classList.toggle("text-white", !!data.live);
      badge.classList.toggle("bg-black/20", !data.live);
      badge.classList.toggle("text-white/60", !data.live);
    }
    text(".js-stream-label", data.live ? "Stream up" : "Stream down");

    var song = el(".js-np-song");
    if (song) {
      song.textContent = data.has_song ? data.song : "No track metadata";
      song.classList.toggle("text-white/60", !data.has_song);
    }
    text(".js-np-artist", data.has_song ? data.artist || "" : "");

    var cover = el(".js-np-cover");
    var icon = el(".js-np-icon");
    if (cover && data.cover_url && cover.getAttribute("src") !== data.cover_url) {
      cover.src = data.cover_url;
    }
    show(cover, !!data.cover_url);
    show(icon, !data.cover_url);
  }

  function applyListeners(data) {
    text(".js-np-listeners", data.listeners);
    // The peak line is server-rendered as hidden when the day has no samples yet;
    // the first poll that finds one reveals it.
    var peak = el(".js-np-peak");
    if (peak) {
      text(".js-np-peak-value", data.peak_today);
      show(peak, data.peak_today > 0);
    }
  }

  function get(url, apply) {
    fetch(url, { headers: { "Accept": "application/json" } })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (data) { if (data) apply(data); })
      .catch(function () {});
  }

  function poll() {
    get("/api/schedule/current", applySchedule);
    get("/api/nowplaying", applyNowPlaying);
    get("/admin/api/listeners", applyListeners);
  }

  poll();
  setInterval(poll, POLL_MS);
})();

// Hover or tap a bar in the Listeners or News-arriving chart to read its value. One
// tooltip serves both: it follows the cursor on hover and appears at the tap point on
// touch (where there is no hover), and the active column is highlighted. A separate IIFE
// from the on-air strip above: it runs wherever bars exist, not just when the card does.
(function () {
  "use strict";

  if (!document.querySelector(".js-bar")) return;

  // A single tooltip on <body>, position:fixed, so clientX/clientY place it directly -
  // no chart-wrapper maths and nothing to clip it. pointer-events-none so it never eats
  // the pointer.
  var tip = document.createElement("div");
  tip.className =
    "js-bar-tip fixed z-50 hidden pointer-events-none -translate-x-1/2 -translate-y-full rounded-md bg-gray-900 px-2 py-1 text-[11px] font-medium text-white shadow-lg whitespace-nowrap";
  document.body.appendChild(tip);

  var activeBar = null;

  function highlight(bar) {
    if (activeBar === bar) return;
    if (activeBar) activeBar.classList.remove("is-active");
    activeBar = bar;
    if (bar) bar.classList.add("is-active");
  }

  function hide() {
    tip.classList.add("hidden");
    highlight(null);
  }

  function showAt(bar, x, y) {
    var text = bar.getAttribute("data-tip");
    if (!text) {
      hide();
      return;
    }
    tip.textContent = text;
    // Centred just above the cursor (translate classes handle the -50%/-100% shift).
    tip.style.left = x + "px";
    tip.style.top = y - 12 + "px";
    tip.classList.remove("hidden");
    highlight(bar);
  }

  // Hover is mouse-only; touch drags would otherwise fight the tap handler below.
  document.addEventListener("pointermove", function (e) {
    if (e.pointerType && e.pointerType !== "mouse") return;
    var bar = e.target.closest(".js-bar");
    if (bar) showAt(bar, e.clientX, e.clientY);
    else hide();
  });

  // Click covers a touch tap: the tooltip then stays (no hover to dismiss it) until the
  // next tap - on another bar it moves, on empty space it hides.
  document.addEventListener("click", function (e) {
    var bar = e.target.closest(".js-bar");
    if (bar) showAt(bar, e.clientX, e.clientY);
    else hide();
  });

  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") hide();
  });
  document.addEventListener("mouseleave", hide);
  window.addEventListener("scroll", hide, true);
})();
