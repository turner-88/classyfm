// Adds .is-scrolled to the sticky header once the page has moved off the top, so
// it can pick up an opaque background and a shadow instead of sitting flat over
// the content. Turbo re-executes this body <script> tag on every in-site
// navigation, so the scroll listener is registered once (guarded by
// window.__headerScrollInit) while the class is re-synced on every visit — the
// header element itself is replaced by Turbo, and a restored scroll position
// fires no scroll event of its own.
(function () {
  "use strict";

  function sync() {
    var header = document.getElementById("site-header");
    if (header) header.classList.toggle("is-scrolled", window.scrollY > 8);
  }

  sync();

  if (window.__headerScrollInit) return;
  window.__headerScrollInit = true;

  window.addEventListener("scroll", sync, { passive: true });
})();
