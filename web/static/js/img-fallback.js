(function () {
  document.querySelectorAll("img.js-img-fade, iframe.js-img-fade").forEach(function (el) {
    if (el.dataset.fallbackBound) return;
    el.dataset.fallbackBound = "1";

    function reveal() {
      el.classList.remove("opacity-0");
    }
    function fail() {
      el.remove();
    }

    // Iframes have no complete/naturalWidth semantics and must not be removed on
    // error — just fade the frame in once it loads. The timeout is the safety net
    // for when this deferred script binds after a cross-origin frame's load event
    // already fired (frames expose no reliable "already loaded" flag); reveal() is
    // idempotent, so load + timeout both firing is harmless.
    if (el.tagName === "IFRAME") {
      el.addEventListener("load", reveal);
      setTimeout(reveal, 1500);
      return;
    }

    if (el.complete) {
      if (el.naturalWidth > 0) reveal();
      else fail();
      return;
    }
    el.addEventListener("load", reveal);
    el.addEventListener("error", fail);
  });
})();
