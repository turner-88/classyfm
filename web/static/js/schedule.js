// Keeps the "Today on air" timeline (today-programs.html) live: polls
// /api/schedule/today and moves each row between its on-air / past / upcoming
// state without a page reload. Rows are rendered server-side in stable
// chronological order and never added, removed, or reordered client-side, so
// matching the poll response to rows by array index is safe - only
// on_air/ended/progress change during a day, not the list itself. Turbo
// re-executes this body <script> tag on every in-site navigation (see radio.js),
// so the setInterval loop is guarded the same way.
(function () {
  "use strict";

  if (!document.querySelector(".js-schedule-list")) return;

  var POLL_MS = 30000;

  // A row's appearance is defined entirely by one state class - is-onair,
  // is-past, or neither (upcoming). Every colour lives in the .schedule-* block
  // in tailwind.css, so this poll cannot disagree with what the server rendered
  // the way an earlier version did, when it re-coloured rows with a different
  // palette than the template used and never cleaned up a slot that had ended.
  function apply(row, s) {
    row.classList.toggle("is-onair", !!s.on_air);
    row.classList.toggle("is-past", !!s.ended);
    // Only the single current slot keeps a colored thumbnail (see tailwind.css).
    row.classList.toggle("is-current", !!s.is_current);

    var badge = row.querySelector(".js-onair-badge");
    var track = row.querySelector(".js-progress-track");
    var bar = row.querySelector(".js-progress-bar");

    if (badge) badge.classList.toggle("hidden", !s.on_air);
    if (track) track.classList.toggle("hidden", !s.on_air);
    if (bar) {
      // Width is set via a `w-[N%]` utility class (safelisted for N=0..100 in
      // tailwind.config.js) rather than an inline style, so it works under a
      // CSP with no `style-src 'unsafe-inline'`.
      bar.className = bar.className.replace(/\bw-\[\d+%\]/, "w-[" + s.progress + "%]");
    }
  }

  // Open the rail at the slot that is on air (or the next one up, during dead
  // air) instead of at 06:00. Assigning scrollTop on the rail itself is what
  // keeps this contained: scrollIntoView() would scroll the page too and yank a
  // visitor away from Home's hero. Runs once per page load - repeating it on
  // every poll would fight whatever the visitor had scrolled to.
  function revealOnAir() {
    document.querySelectorAll(".js-schedule-scroll").forEach(function (box) {
      var fade = box.parentNode.querySelector(".js-schedule-fade");
      if (fade) fade.classList.toggle("hidden", box.scrollHeight <= box.clientHeight);

      var row = box.querySelector(".js-schedule-row.is-onair") ||
                box.querySelector(".js-schedule-row:not(.is-past)");
      // offsetTop is measured against the rail because it carries `relative`.
      if (row) box.scrollTop = Math.max(0, row.offsetTop - 12);
    });
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

  revealOnAir();
  poll();

  if (window.__scheduleInit) return;
  window.__scheduleInit = true;
  setInterval(poll, POLL_MS);
})();
