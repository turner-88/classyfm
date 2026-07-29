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

  var FOCUSABLE = 'a[href], button:not([disabled]), summary, [tabindex]:not([tabindex="-1"])';

  function isOpen() {
    return !sidebar.classList.contains("-translate-x-full");
  }

  function setOpen(open) {
    sidebar.classList.toggle("-translate-x-full", !open);
    backdrop.classList.toggle("hidden", !open);
    toggle.setAttribute("aria-expanded", open ? "true" : "false");
    toggle.querySelectorAll("[data-header-menu-icon]").forEach(function (icon) {
      icon.classList.toggle("hidden", icon.getAttribute("data-header-menu-icon") === "open" ? open : !open);
    });
    // Closed, the drawer is only translated off-screen, so without inert its
    // links stay in the tab order. Open, keep focus inside it and hand focus
    // back to the hamburger on close.
    sidebar.inert = !open;
    if (open) {
      var first = sidebar.querySelector(FOCUSABLE);
      if (first) first.focus();
    } else if (sidebar.contains(document.activeElement)) {
      toggle.focus();
    }
  }

  toggle.addEventListener("click", function () {
    setOpen(!isOpen());
  });

  // Focus trap: Tab off either end of the drawer wraps to the other end.
  sidebar.addEventListener("keydown", function (e) {
    if (e.key !== "Tab" || !isOpen()) return;
    var items = Array.prototype.filter.call(sidebar.querySelectorAll(FOCUSABLE), function (el) {
      return el.offsetParent !== null;
    });
    if (items.length === 0) return;
    var first = items[0];
    var last = items[items.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
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

  setOpen(false);
})();
