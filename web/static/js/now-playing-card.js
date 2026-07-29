// Keeps Home's "On Air" poster card (inline in public/home.html)
// live: polls /api/schedule/current every 30s and updates the single card in
// place - including swapping to the next on-air program - without a page
// reload. Independent of schedule.js/today-programs.html, which serve Live's
// full-day list. Turbo re-executes this body <script> tag on every in-site
// navigation (see radio.js), so the setInterval loop is guarded the same way.
(function () {
  "use strict";

  var wrap = document.querySelector(".js-now-card-wrap");
  if (!wrap) return;

  var POLL_MS = 30000;
  var card = wrap.querySelector(".js-now-card");
  var empty = wrap.querySelector(".js-now-empty");

  function setImage(img, src) {
    if (img.src === src) return;
    img.classList.add("opacity-0");
    img.src = src;
    img.onload = function () { img.classList.remove("opacity-0"); };
    img.onerror = function () { img.remove(); };
  }

  function apply(data) {
    if (!data.on_air) {
      if (card) card.classList.add("hidden");
      if (empty) empty.classList.remove("hidden");
      return;
    }
    if (empty) empty.classList.add("hidden");
    if (!card) return;
    card.classList.remove("hidden");

    card.href = "/program/" + data.slug;

    var title = card.querySelector(".js-schedule-title");
    if (title) title.textContent = data.title;

    var host = card.querySelector(".js-schedule-host");
    if (host) host.textContent = data.host || "";

    var time = card.querySelector(".js-schedule-time");
    if (time) time.textContent = data.start + "–" + data.end;

    var bar = card.querySelector(".js-progress-bar");
    if (bar) bar.className = bar.className.replace(/\bw-\[\d+%\]/, "w-[" + data.progress + "%]");

    var img = card.querySelector("img");
    if (img && data.image) setImage(img, data.image);
  }

  function poll() {
    fetch("/api/schedule/current", { headers: { "Accept": "application/json" } })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (data) { if (data) apply(data); })
      .catch(function () {});
  }

  poll();

  if (window.__nowPlayingCardInit) return;
  window.__nowPlayingCardInit = true;
  setInterval(poll, POLL_MS);
})();
