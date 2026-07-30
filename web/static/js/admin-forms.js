// Confirmation prompts for destructive admin actions.
//
// These used to be inline `onsubmit="return confirm(...)"` attributes, which the
// CSP in internal/middleware/security.go silently blocks - script-src is 'self'
// with no 'unsafe-inline', and that covers event-handler attributes as well as
// <script> blocks, so every delete fired straight through without asking. The
// listener is delegated from the document so it also covers rows rendered into
// tables after load.
(function () {
  "use strict";

  document.addEventListener("click", function (e) {
    var el = e.target.closest("[data-confirm]");
    if (!el) return;
    if (!window.confirm(el.getAttribute("data-confirm"))) {
      e.preventDefault();
    }
  });
})();
