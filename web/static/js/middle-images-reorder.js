// Reorders the Hot Release mid-article gallery in the admin editor. The stored
// order is simply the DOM order of the [name=existing_middle_image] hidden inputs
// (a form serializes fields top-to-bottom), so moving a row *is* the reorder -
// there is no separate position field to keep in sync.
//
// Two mechanisms, same effect: drag-and-drop for the mouse, and up/down arrow
// buttons for keyboard and touch. Bound directly to the elements found at load;
// Turbo swaps the whole <body> and re-runs this on every navigation, so the old
// nodes (and their listeners) are discarded and there is nothing to clean up.
(function () {
  "use strict";

  document.querySelectorAll("[data-middle-images]").forEach(function (list) {
    var dragging = null;

    list.querySelectorAll("[data-middle-image-row]").forEach(function (row) {
      row.addEventListener("dragstart", function () {
        dragging = row;
        row.classList.add("opacity-50");
      });
      row.addEventListener("dragend", function () {
        row.classList.remove("opacity-50");
        dragging = null;
      });
    });

    list.addEventListener("dragover", function (e) {
      if (!dragging) return;
      e.preventDefault();
      var after = rowAfter(list, e.clientY);
      if (after == null) list.appendChild(dragging);
      else list.insertBefore(dragging, after);
    });

    list.querySelectorAll("[data-move-up]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var row = btn.closest("[data-middle-image-row]");
        var prev = row && row.previousElementSibling;
        if (prev) list.insertBefore(row, prev);
      });
    });

    list.querySelectorAll("[data-move-down]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var row = btn.closest("[data-middle-image-row]");
        var next = row && row.nextElementSibling;
        if (next) list.insertBefore(next, row);
      });
    });
  });

  // The row whose vertical midpoint sits just below the pointer - the one the
  // dragged row should be inserted before. null means past the last row (append).
  function rowAfter(list, y) {
    var rows = Array.prototype.slice.call(
      list.querySelectorAll("[data-middle-image-row]:not(.opacity-50)")
    );
    var closest = { offset: Number.NEGATIVE_INFINITY, element: null };
    rows.forEach(function (row) {
      var box = row.getBoundingClientRect();
      var offset = y - box.top - box.height / 2;
      if (offset < 0 && offset > closest.offset) {
        closest = { offset: offset, element: row };
      }
    });
    return closest.element;
  }
})();
