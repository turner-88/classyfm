// Reading progress bar and copy-link button on a news article. Both bail out
// immediately on every other page, so this costs nothing where it isn't used.
//
// Turbo re-executes this body <script> tag on every in-site navigation. The
// copy-link listener is bound to an element Turbo replaces along with the page,
// so it needs no guard; the scroll/resize listeners are on window and would
// stack up, so they are registered once (window.__newsProgressInit) while the
// bar is re-synced on every visit - a Turbo-restored scroll position fires no
// scroll event of its own.
(function () {
  "use strict";

  // Progress bar.
  function syncProgress() {
    var bar = document.querySelector(".js-read-progress-bar");
    if (!bar) return;

    // Everything below the fold is what's left to read; a page shorter than the
    // viewport has nothing to track and reads as complete.
    var scrollable = document.documentElement.scrollHeight - window.innerHeight;
    var pct = scrollable > 0 ? Math.round((window.scrollY / scrollable) * 100) : 100;
    pct = Math.min(100, Math.max(0, pct));

    // Width is set via a `w-[N%]` utility class (safelisted for N=0..100 in
    // tailwind.config.js) rather than an inline style, so it works under a CSP
    // with no `style-src 'unsafe-inline'` - same as schedule.js.
    bar.className = bar.className.replace(/\bw-\[\d+%\]/, "w-[" + pct + "%]");
  }

  syncProgress();

  if (!window.__newsProgressInit) {
    window.__newsProgressInit = true;
    window.addEventListener("scroll", syncProgress, { passive: true });
    window.addEventListener("resize", syncProgress, { passive: true });
  }

  // Copy link. Left inert where the Clipboard API is missing (any insecure
  // origin) rather than hidden - the three share links beside it still work, and
  // a button that silently does nothing is worse than one that was never armed.
  var copyBtn = document.querySelector(".js-copy-link");
  if (!copyBtn || !navigator.clipboard) return;

  var label = copyBtn.querySelector(".js-copy-label");
  var idleIcon = copyBtn.querySelector('[data-copy="idle"]');
  var doneIcon = copyBtn.querySelector('[data-copy="done"]');
  var original = label ? label.textContent : "";
  var resetTimer = null;

  function state(copied) {
    if (label) label.textContent = copied ? "Copied" : original;
    if (idleIcon) idleIcon.classList.toggle("hidden", copied);
    if (doneIcon) doneIcon.classList.toggle("hidden", !copied);
  }

  copyBtn.addEventListener("click", function () {
    // Nothing on failure: the URL is in the address bar either way, and an
    // error state on a convenience button is noise.
    navigator.clipboard.writeText(copyBtn.dataset.url).then(function () {
      state(true);
      clearTimeout(resetTimer);
      resetTimer = setTimeout(function () {
        state(false);
      }, 2000);
    });
  });
})();
