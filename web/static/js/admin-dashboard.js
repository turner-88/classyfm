// Keeps the admin dashboard's on-air strip live: polls the same public endpoints
// the site itself uses (/api/schedule/current for the program, /api/nowplaying for
// the track and stream state) and updates the card in place. No admin-only endpoint
// exists for this on purpose - the data is public either way, and duplicating it
// would mean two things to keep in step.
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

  function get(url, apply) {
    fetch(url, { headers: { "Accept": "application/json" } })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (data) { if (data) apply(data); })
      .catch(function () {});
  }

  function poll() {
    get("/api/schedule/current", applySchedule);
    get("/api/nowplaying", applyNowPlaying);
  }

  poll();
  setInterval(poll, POLL_MS);
})();
