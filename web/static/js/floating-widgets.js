// Collapse/restore for the floating widgets stacked at the bottom-right
// (partials/floating-widgets.html): each .js-widget holds a .js-widget-card plus the
// .js-widget-bubble it collapses into, and exactly one of the two is ever visible.
//
// State is per page load only - no localStorage, no sessionStorage. #floating-stack
// is data-turbo-permanent, so Turbo carries these nodes (and whatever the visitor
// collapsed) across in-site navigation as-is, and a hard reload starts fresh. That
// also means the listeners bound here survive: Turbo re-executes this body <script>
// on every visit against the *same* nodes, so each bind is guarded by data-bound and
// the one-time narrow-screen default by data-init, both of which live on the
// permanent nodes and are therefore carried along with them.
//
// The radio player's <audio> is outside the stack entirely, so collapsing its card
// never interrupts playback - and radio.js keeps the collapsed bubble's play/pause
// icon current through the .js-radio-state hook it carries.
(function () {
  "use strict";

  var stack = document.getElementById("floating-stack");
  if (!stack) return;

  stack.querySelectorAll(".js-widget").forEach(function (widget) {
    var card = widget.querySelector(".js-widget-card");
    var bubble = widget.querySelector(".js-widget-bubble");
    if (!card || !bubble) return;

    function setCollapsed(collapsed) {
      card.classList.toggle("hidden", collapsed);
      bubble.classList.toggle("hidden", !collapsed);
      bubble.setAttribute("aria-expanded", collapsed ? "false" : "true");
    }

    widget.querySelectorAll(".js-widget-close").forEach(function (btn) {
      if (btn.dataset.bound) return;
      btn.dataset.bound = "1";
      btn.addEventListener("click", function () { setCollapsed(true); });
    });

    if (!bubble.dataset.bound) {
      bubble.dataset.bound = "1";
      bubble.addEventListener("click", function () { setCollapsed(false); });
    }

    // A phone has room for one card, not two. Widgets marked data-collapse-below-sm
    // start as a bubble on a narrow screen - once per page load, so re-opening one
    // and then navigating doesn't fold it back up under the visitor.
    if (!widget.dataset.init) {
      widget.dataset.init = "1";
      if (widget.hasAttribute("data-collapse-below-sm") &&
          window.matchMedia("(max-width: 639px)").matches) {
        setCollapsed(true);
      }
    }
  });
})();
