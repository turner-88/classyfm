// Mobile-only admin nav: slides the sidebar in/out with a backdrop and swaps
// the hamburger/close icon. Desktop uses the sidebar in-flow (md:sticky),
// this script only toggles the transform/backdrop classes that matter below
// the md breakpoint.
(function () {
  "use strict";

  var toggle = document.querySelector("[data-admin-menu-toggle]");
  var sidebar = document.getElementById("admin-sidebar");
  var backdrop = document.querySelector("[data-admin-menu-backdrop]");
  if (!toggle || !sidebar || !backdrop) return;

  function setOpen(open) {
    sidebar.classList.toggle("-translate-x-full", !open);
    backdrop.classList.toggle("hidden", !open);
    toggle.setAttribute("aria-expanded", open ? "true" : "false");
    toggle.querySelectorAll("[data-admin-menu-icon]").forEach(function (icon) {
      icon.classList.toggle("hidden", icon.getAttribute("data-admin-menu-icon") === "open" ? open : !open);
    });
  }

  toggle.addEventListener("click", function () {
    setOpen(sidebar.classList.contains("-translate-x-full"));
  });

  backdrop.addEventListener("click", function () {
    setOpen(false);
  });

  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") setOpen(false);
  });
})();
