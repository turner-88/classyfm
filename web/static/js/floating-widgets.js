// Collapse/restore for the floating widgets stacked at the bottom-right
// (partials/floating-widgets.html): each .js-widget holds a .js-widget-card plus the
// .js-widget-bubble it collapses into, and exactly one of the two is ever visible.
//
// The collapsed choice persists per widget (keyed by data-widget-key) in localStorage
// as "classyfm.widget.<key>.collapsed", so a hard reload restores it. #floating-stack
// is data-turbo-permanent, so Turbo also carries these nodes (and whatever the visitor
// collapsed) across in-site navigation as-is; the stored value is only read on a fresh
// load (guarded by data-init below), which is exactly the hard-reload case. Only an
// explicit toggle is persisted - the narrow-screen default is applied without writing,
// so a responsive collapse never becomes a sticky preference. That
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

    var key = widget.dataset.widgetKey
      ? "classyfm.widget." + widget.dataset.widgetKey + ".collapsed"
      : null;

    function setCollapsed(collapsed, persist) {
      card.classList.toggle("hidden", collapsed);
      bubble.classList.toggle("hidden", !collapsed);
      bubble.setAttribute("aria-expanded", collapsed ? "false" : "true");
      if (persist && key) {
        try { localStorage.setItem(key, collapsed ? "1" : "0"); } catch (e) {}
      }
    }

    widget.querySelectorAll(".js-widget-close").forEach(function (btn) {
      if (btn.dataset.bound) return;
      btn.dataset.bound = "1";
      btn.addEventListener("click", function () { setCollapsed(true, true); });
    });

    if (!bubble.dataset.bound) {
      bubble.dataset.bound = "1";
      bubble.addEventListener("click", function () { setCollapsed(false, true); });
    }

    // Once per page load (so re-opening one and then navigating doesn't fold it back
    // up under the visitor): a stored choice from a prior visit wins; otherwise a
    // phone has room for one card, not two, so widgets marked data-collapse-below-sm
    // start as a bubble on a narrow screen. Neither branch persists - see the header.
    if (!widget.dataset.init) {
      widget.dataset.init = "1";
      var stored = null;
      if (key) { try { stored = localStorage.getItem(key); } catch (e) {} }
      if (stored === "1" || stored === "0") {
        setCollapsed(stored === "1", false);
      } else if (widget.hasAttribute("data-collapse-below-sm") &&
          window.matchMedia("(max-width: 639px)").matches) {
        setCollapsed(true, false);
      }
    }
  });
})();
