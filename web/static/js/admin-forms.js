// Admin form behaviour: destructive-action confirmation, repeating rows, and an
// unsaved-changes guard.
//
// Everything is a delegated listener on the document. Inline handlers such as
// `onsubmit="return confirm(...)"` are silently blocked by the CSP in
// internal/middleware/security.go - script-src is 'self' with no 'unsafe-inline',
// and that covers event-handler attributes as well as <script> blocks - so every
// delete used to fire straight through without asking. Delegation also covers rows
// cloned in after load.
(function () {
  "use strict";

  document.addEventListener("click", function (e) {
    var el = e.target.closest("[data-confirm]");
    if (!el) return;
    if (!window.confirm(el.getAttribute("data-confirm"))) {
      e.preventDefault();
    }
  });

  // --- repeating rows -------------------------------------------------------
  // A row list is a [data-repeat] container holding a [data-repeat-body] and a
  // <template data-repeat-template>. The template's field names carry the literal
  // token __KEY__, which is swapped for a unique key per added row: a row's
  // multi-valued field can't ride in a parallel array (its checkboxes post a
  // variable number of values), so it is named slot_bc_<key> and paired back up
  // server-side via the row's slot_key.
  var rowSeq = 0;

  document.addEventListener("click", function (e) {
    var addBtn = e.target.closest("[data-repeat-add]");
    if (addBtn) {
      e.preventDefault();
      addRow(addBtn.closest("[data-repeat]"));
      return;
    }
    var removeBtn = e.target.closest("[data-repeat-remove]");
    if (removeBtn) {
      e.preventDefault();
      removeRow(removeBtn);
    }
  });

  function addRow(list) {
    if (!list) return;
    var tpl = list.querySelector("[data-repeat-template]");
    var body = list.querySelector("[data-repeat-body]");
    if (!tpl || !body) return;

    var key = "new-" + rowSeq++;
    var frag = tpl.content.cloneNode(true);
    ["name", "id", "for", "aria-labelledby"].forEach(function (attr) {
      frag.querySelectorAll("[" + attr + "*='__KEY__']").forEach(function (el) {
        el.setAttribute(attr, el.getAttribute(attr).split("__KEY__").join(key));
      });
    });

    var empty = list.querySelector("[data-repeat-empty]");
    if (empty) empty.remove();

    var row = frag.firstElementChild;
    body.appendChild(frag);
    markDirty(list.closest("form"));
    // Widgets that need per-element setup (admin-multiselect.js) can't rely on
    // their own load-time pass for a row that appears afterwards.
    if (row) row.dispatchEvent(new CustomEvent("repeat:rowadded", { bubbles: true }));
    var first = row && row.querySelector("select, input, textarea, button");
    if (first) first.focus();
  }

  function removeRow(btn) {
    var row = btn.closest("[data-repeat-row]");
    if (!row) return;
    var list = row.closest("[data-repeat]");
    var form = row.closest("form");
    row.remove();
    markDirty(form);
    // A removed row's fields simply stop being posted; the server deletes any
    // stored row whose id is missing from the submission.
    var body = list && list.querySelector("[data-repeat-body]");
    if (body && !body.querySelector("[data-repeat-row]")) {
      var msg = list.getAttribute("data-repeat-empty-text");
      if (msg) {
        var p = document.createElement("p");
        p.className = "repeat-empty";
        p.setAttribute("data-repeat-empty", "");
        p.textContent = msg;
        body.appendChild(p);
      }
    }
  }

  // --- conditional panels ---------------------------------------------------
  // A [data-toggle-group="<radio name>"] container shows only those descendant
  // [data-toggle-case="a b"] panels whose case list contains the value of the
  // group's checked radio. The markup ships with every panel visible so a JS
  // failure degrades to the old always-show-everything form rather than to a
  // form with no fields at all.
  document.addEventListener("change", function (e) {
    var radio = e.target;
    if (!radio.matches || !radio.matches("input[type=radio]")) return;
    var group = radio.closest("[data-toggle-group]");
    if (group && group.getAttribute("data-toggle-group") === radio.name) {
      applyToggleGroup(group);
    }
  });

  function applyToggleGroup(group) {
    var name = group.getAttribute("data-toggle-group");
    var checked = group.querySelector("input[type=radio][name='" + name + "']:checked");
    var value = checked ? checked.value : null;
    group.querySelectorAll("[data-toggle-case]").forEach(function (panel) {
      var on =
        value !== null &&
        panel.getAttribute("data-toggle-case").split(/\s+/).indexOf(value) !== -1;
      panel.classList.toggle("hidden", !on);
      // A hidden input is still validated on submit, and the browser refuses to
      // report a violation it cannot scroll to - a stale invalid URL in the
      // unselected panel would block the save with nothing on screen to explain
      // it. Disabling also keeps the unselected media out of the submission, so
      // the server knows to leave its stored value alone.
      panel.querySelectorAll("input, select, textarea").forEach(function (field) {
        field.disabled = !on;
      });
    });
  }

  document.querySelectorAll("[data-toggle-group]").forEach(applyToggleGroup);

  // --- unsaved-changes guard ------------------------------------------------
  // Removing a row is now client-side and unsaved until the page's one Save, so a
  // stray back-navigation would discard it with nothing to show for it.
  var dirtyForms = new WeakSet();

  function markDirty(form) {
    if (form && form.hasAttribute("data-dirty-guard")) dirtyForms.add(form);
  }

  document.addEventListener("input", function (e) {
    // Filter and search boxes sit inside the form for layout reasons but post
    // nothing, so typing in one is not an unsaved change.
    if (e.target.hasAttribute("data-no-dirty")) return;
    markDirty(e.target.form);
  });
  document.addEventListener("change", function (e) {
    markDirty(e.target.form);
  });
  document.addEventListener(
    "submit",
    function (e) {
      dirtyForms.delete(e.target);
    },
    true
  );

  window.addEventListener("beforeunload", function (e) {
    var forms = document.querySelectorAll("form[data-dirty-guard]");
    for (var i = 0; i < forms.length; i++) {
      if (dirtyForms.has(forms[i])) {
        e.preventDefault();
        e.returnValue = "";
        return;
      }
    }
  });
})();
