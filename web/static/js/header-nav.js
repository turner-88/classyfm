// Mobile-only public nav: slides the header sidebar in/out with a backdrop and
// swaps the hamburger/close icon, mirroring admin-nav.js. Unlike the admin
// sidebar, this one is inside Turbo-managed page content, so it's a fresh DOM
// subtree on every visit - no persistence guard needed, this whole script just
// re-runs and re-binds against whatever is currently in the document.
(function () {
  "use strict";

  var toggle = document.querySelector("[data-header-menu-toggle]");
  var sidebar = document.getElementById("header-sidebar");
  var backdrop = document.querySelector("[data-header-menu-backdrop]");
  var closeBtn = document.querySelector("[data-header-menu-close]");
  if (!toggle || !sidebar || !backdrop) return;

  function setOpen(open) {
    sidebar.classList.toggle("-translate-x-full", !open);
    backdrop.classList.toggle("hidden", !open);
    toggle.setAttribute("aria-expanded", open ? "true" : "false");
    toggle.querySelectorAll("[data-header-menu-icon]").forEach(function (icon) {
      icon.classList.toggle("hidden", icon.getAttribute("data-header-menu-icon") === "open" ? open : !open);
    });
  }

  toggle.addEventListener("click", function () {
    setOpen(sidebar.classList.contains("-translate-x-full"));
  });

  backdrop.addEventListener("click", function () {
    setOpen(false);
  });

  if (closeBtn) {
    closeBtn.addEventListener("click", function () {
      setOpen(false);
    });
  }

  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") setOpen(false);
  });
})();
