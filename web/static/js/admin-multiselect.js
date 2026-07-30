// A single-line multi-select: one .form-select-shaped trigger showing the picked
// names, and a dropdown panel of ordinary checkboxes with a filter box.
//
// The options are real <input type="checkbox"> sharing one name, so the server
// reads them exactly as it read the <select multiple> this replaces, and nothing
// here has to serialise anything. The panel starts open in the markup and is
// collapsed here at init: if this script fails to run the field degrades to a
// plain visible checkbox list rather than to a button that does nothing.
//
// Positioning is entirely in CSS (.multiselect-panel is absolute inside a
// relative .multiselect). Nothing writes element.style - the CSP in
// internal/middleware/security.go has no style-src 'unsafe-inline', and the rest
// of this codebase keeps to toggling classes for the same reason.
//
// Everything is delegated off the document so rows cloned in by admin-forms.js
// after load are covered without re-initialising. That file rewrites only the
// name/id/for/aria-labelledby attributes of a cloned row, so a widget must never
// carry its row key anywhere else: the parts here find each other by walking up
// to the nearest [data-multiselect].
(function () {
  "use strict";

  function panelOf(root) {
    return root.querySelector("[data-multiselect-panel]");
  }

  function isOpen(root) {
    var panel = panelOf(root);
    return !!panel && !panel.classList.contains("hidden");
  }

  function setOpen(root, open) {
    var panel = panelOf(root);
    var toggle = root.querySelector("[data-multiselect-toggle]");
    if (!panel) return;
    panel.classList.toggle("hidden", !open);
    if (toggle) toggle.setAttribute("aria-expanded", open ? "true" : "false");
  }

  function closeAll(except) {
    document.querySelectorAll("[data-multiselect]").forEach(function (root) {
      if (root !== except) setOpen(root, false);
    });
  }

  // Rewrites the trigger's summary line from whatever is currently checked.
  function sync(root) {
    var value = root.querySelector("[data-multiselect-value]");
    if (!value) return;
    var names = [];
    root.querySelectorAll("[data-multiselect-option] input:checked").forEach(function (cb) {
      names.push(cb.closest("[data-multiselect-option]").textContent.trim());
    });
    var text = names.join(", ");
    // Truncated to one line by CSS, so the full set lives in the tooltip.
    value.textContent = text || root.getAttribute("data-multiselect-placeholder") || "";
    value.classList.toggle("is-empty", names.length === 0);
    var toggle = root.querySelector("[data-multiselect-toggle]");
    if (toggle) {
      if (text) toggle.setAttribute("title", text);
      else toggle.removeAttribute("title");
    }
  }

  function filter(root, query) {
    var q = query.trim().toLowerCase();
    var shown = 0;
    root.querySelectorAll("[data-multiselect-option]").forEach(function (opt) {
      var cb = opt.querySelector("input");
      // A checked option always stays visible: filtering something out of reach
      // would leave a selection the admin can see in the trigger but can't undo.
      var match = !q || (cb && cb.checked) || opt.textContent.toLowerCase().indexOf(q) !== -1;
      opt.classList.toggle("hidden", !match);
      if (match) shown++;
    });
    var none = root.querySelector("[data-multiselect-noresult]");
    if (none) none.classList.toggle("hidden", shown !== 0);
  }

  function init(root) {
    setOpen(root, false);
    sync(root);
  }

  document.querySelectorAll("[data-multiselect]").forEach(init);

  // A schedule row cloned in by admin-forms.js missed the pass above: its panel
  // is still the open list the markup ships and its trigger is still blank.
  document.addEventListener("repeat:rowadded", function (e) {
    e.target.querySelectorAll("[data-multiselect]").forEach(init);
  });

  document.addEventListener("click", function (e) {
    var toggle = e.target.closest("[data-multiselect-toggle]");
    if (!toggle) {
      // A click anywhere outside a widget dismisses whatever is open; a click
      // inside the panel must not, or ticking a box would close it.
      if (!e.target.closest("[data-multiselect]")) closeAll(null);
      return;
    }
    var root = toggle.closest("[data-multiselect]");
    var open = !isOpen(root);
    closeAll(root);
    setOpen(root, open);
    if (open) {
      var panel = panelOf(root);
      // A panel hanging off one of the lower schedule rows can open below the
      // fold; nudge it into view rather than leaving an apparently empty click.
      if (panel) panel.scrollIntoView({ block: "nearest" });
      var search = root.querySelector("[data-multiselect-search]");
      if (search) {
        search.value = "";
        filter(root, "");
        search.focus();
      }
    }
  });

  document.addEventListener("change", function (e) {
    var opt = e.target.closest("[data-multiselect-option]");
    if (opt) sync(opt.closest("[data-multiselect]"));
  });

  document.addEventListener("input", function (e) {
    var search = e.target.closest("[data-multiselect-search]");
    if (search) filter(search.closest("[data-multiselect]"), search.value);
  });

  document.addEventListener("keydown", function (e) {
    var root = e.target.closest("[data-multiselect]");
    if (e.key === "Escape") {
      if (!root || !isOpen(root)) return;
      setOpen(root, false);
      var toggle = root.querySelector("[data-multiselect-toggle]");
      if (toggle) toggle.focus();
      return;
    }
    // The filter box sits inside the page's one big form, where Enter would
    // otherwise submit the whole program.
    if (e.key === "Enter" && e.target.closest("[data-multiselect-search]")) {
      e.preventDefault();
    }
  });
})();
