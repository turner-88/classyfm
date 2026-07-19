// Keeps "Program Hari Ini" (today-programs.html) live: polls /api/schedule/today
// and toggles each row's on-air styling/progress bar without a page reload. Rows
// are rendered server-side in stable chronological order and never added,
// removed, or reordered client-side, so matching the poll response to rows by
// array index is safe - only on_air/progress change during a day, not the list
// itself. Turbo re-executes this body <script> tag on every in-site navigation
// (see radio.js), so the setInterval loop is guarded the same way.
(function () {
  "use strict";

  if (!document.querySelector(".js-schedule-list")) return;

  var POLL_MS = 30000;

  function apply(row, s) {
    var badge = row.querySelector(".js-onair-badge");
    var track = row.querySelector(".js-progress-track");
    var bar = row.querySelector(".js-progress-bar");
    var title = row.querySelector(".js-schedule-title");
    var time = row.querySelector(".js-schedule-time");

    row.classList.toggle("border-brand/30", s.on_air);
    row.classList.toggle("bg-brand/5", s.on_air);
    row.classList.toggle("shadow-sm", s.on_air);
    row.classList.toggle("border-gray-200", !s.on_air);
    row.classList.toggle("hover:border-gray-300", !s.on_air);

    if (badge) badge.classList.toggle("hidden", !s.on_air);
    if (track) track.classList.toggle("hidden", !s.on_air);
    if (bar) {
      // Width is set via a `w-[N%]` utility class (safelisted for N=0..100 in
      // tailwind.config.js) rather than an inline style, so it works under a
      // CSP with no `style-src 'unsafe-inline'`.
      bar.className = bar.className.replace(/\bw-\[\d+%\]/, "w-[" + s.progress + "%]");
    }

    if (title) {
      title.classList.toggle("text-gray-900", s.on_air);
      title.classList.toggle("text-gray-600", !s.on_air);
    }
    if (time) {
      time.classList.toggle("text-brand", s.on_air);
      time.classList.toggle("text-gray-400", !s.on_air);
    }
  }

  function poll() {
    var lists = document.querySelectorAll(".js-schedule-list");
    if (!lists.length) return;
    fetch("/api/schedule/today", { headers: { "Accept": "application/json" } })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (states) {
        if (!states) return;
        lists.forEach(function (list) {
          list.querySelectorAll(".js-schedule-row").forEach(function (row, i) {
            if (states[i]) apply(row, states[i]);
          });
        });
      })
      .catch(function () {});
  }

  poll();

  if (window.__scheduleInit) return;
  window.__scheduleInit = true;
  setInterval(poll, POLL_MS);
})();
