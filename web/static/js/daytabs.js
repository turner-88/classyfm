// Day tabs on the Program page. Every day's panel is already in the document
// (server-rendered), so this only moves the `hidden` class and the active chip
// around - no fetching, no rebuilding, and the page still shows today's schedule
// if this script never runs.
//
// Turbo re-executes this body <script> on every in-site navigation, and the
// listeners are bound to elements that are replaced along with the page, so
// there is nothing to guard against re-running (unlike schedule.js's interval).
(function () {
  "use strict";

  var tablist = document.querySelector(".js-daytabs");
  if (!tablist) return;

  var tabs = Array.prototype.slice.call(tablist.querySelectorAll(".js-daytab"));
  if (!tabs.length) return;

  function select(tab) {
    tabs.forEach(function (t) {
      var on = t === tab;
      t.classList.toggle("chip-active", on);
      t.setAttribute("aria-selected", on ? "true" : "false");
    });
    document.querySelectorAll(".js-daypanel").forEach(function (panel) {
      panel.classList.toggle("hidden", panel.dataset.day !== tab.dataset.day);
    });
  }

  tabs.forEach(function (tab, i) {
    tab.addEventListener("click", function () {
      select(tab);
    });

    // Arrow keys walk the strip and wrap around, so the whole week is reachable
    // without tabbing through it a day at a time.
    tab.addEventListener("keydown", function (e) {
      var step = e.key === "ArrowRight" ? 1 : e.key === "ArrowLeft" ? -1 : 0;
      if (!step) return;
      e.preventDefault();
      var next = tabs[(i + step + tabs.length) % tabs.length];
      select(next);
      next.focus();
    });
  });
})();
