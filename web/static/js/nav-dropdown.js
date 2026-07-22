// Progressive enhancement over the native <details>/<summary> desktop nav
// dropdowns (data-nav-dropdown): closes sibling open dropdowns when one opens,
// and closes an open dropdown on outside click or Escape. The page is fully
// functional without this (native <details> toggling still works) - same
// re-run-on-every-Turbo-visit model as header-nav.js, no persistence guard needed.
(function () {
  "use strict";

  var dropdowns = document.querySelectorAll("[data-nav-dropdown]");
  if (!dropdowns.length) return;

  dropdowns.forEach(function (d) {
    d.addEventListener("toggle", function () {
      if (!d.open) return;
      dropdowns.forEach(function (other) {
        if (other !== d) other.open = false;
      });
    });
  });

  document.addEventListener("click", function (e) {
    dropdowns.forEach(function (d) {
      if (d.open && !d.contains(e.target)) d.open = false;
    });
  });

  document.addEventListener("keydown", function (e) {
    if (e.key !== "Escape") return;
    dropdowns.forEach(function (d) {
      d.open = false;
    });
  });
})();
