// Live mode for the admin chat moderation table (/admin/chat). When toggled on, it polls
// the admin JSON feed for messages newer than the ones already on the page and prepends
// them to the top of the table (newest-first, matching the server-rendered order). It is a
// viewing aid only: hide/ban stay plain POST forms that reload the page, so moderation never
// depends on this script. Absent JS, the page is a normal static table.
(function () {
  "use strict";

  var POLL_MS = 5000;

  var root = document.querySelector("[data-chat-root]");
  if (!root) return;

  var tbody = root.querySelector("[data-chat-tbody]");
  var rowTemplate = root.querySelector("[data-chat-row-template]");
  var toggle = root.querySelector("[data-chat-live-toggle]");
  if (!tbody || !rowTemplate || !toggle) return;

  // Optional: manual refresh does a single poll when Live is off; absent, Live still works.
  var refreshBtn = root.querySelector("[data-chat-refresh]");

  var csrf = root.getAttribute("data-chat-csrf") || "";
  var endpoint = root.getAttribute("data-chat-endpoint") || "/admin/chat/messages.json";
  var since = parseInt(root.getAttribute("data-chat-since"), 10) || 0;

  var dot = toggle.querySelector("[data-chat-live-dot]");
  var label = toggle.querySelector("[data-chat-live-label]");
  var timer = null;
  var inFlight = false;

  function setLiveUI(on) {
    toggle.setAttribute("aria-pressed", on ? "true" : "false");
    if (label) label.textContent = on ? "Live: on" : "Live: off";
    if (dot) {
      dot.classList.toggle("bg-green-500", on);
      dot.classList.toggle("bg-gray-300", !on);
      dot.classList.toggle("animate-pulse", on);
    }
    // Live polls continuously, so a manual refresh is redundant while it's on.
    if (refreshBtn) refreshBtn.classList.toggle("hidden", on);
  }

  // buildRow clones the server-owned skeleton and fills it from one message payload.
  function buildRow(m) {
    var frag = rowTemplate.content.cloneNode(true);
    var row = frag.querySelector("[data-chat-row]");
    row.setAttribute("data-chat-id", m.id);
    row.setAttribute("data-chat-user", m.user_id);

    var hideForm = row.querySelector("[data-chat-hide-form]");
    var banForm = row.querySelector("[data-chat-ban-form]");
    hideForm.setAttribute("action", "/admin/chat/messages/" + m.id + "/hide");
    banForm.setAttribute("action", "/admin/chat/users/" + m.user_id + "/ban");
    hideForm.querySelector('input[name="csrf_token"]').value = csrf;
    banForm.querySelector('input[name="csrf_token"]').value = csrf;

    row.querySelector("[data-chat-name]").textContent = m.name || "";
    var badge = row.querySelector("[data-chat-admin-badge]");
    if (badge) badge.classList.toggle("hidden", !m.is_admin);
    row.querySelector("[data-chat-body]").textContent = m.body || "";
    row.querySelector("[data-chat-time]").textContent = m.time || "";
    return row;
  }

  function prepend(messages) {
    if (!messages || !messages.length) return;
    var empty = tbody.querySelector("[data-chat-empty]");
    if (empty) empty.remove();
    // messages come oldest-first (id ASC); prepend each so the newest ends up on top.
    messages.forEach(function (m) {
      if (m.id > since) since = m.id;
      tbody.insertBefore(buildRow(m), tbody.firstChild);
    });
  }

  function poll() {
    if (inFlight) return;
    inFlight = true;
    fetch(endpoint + "?since=" + encodeURIComponent(since), {
      headers: { Accept: "application/json" },
      credentials: "same-origin",
    })
      .then(function (res) { return res.ok ? res.json() : { messages: [] }; })
      .then(function (data) { prepend(data && data.messages); })
      .catch(function () { /* transient; next tick retries */ })
      .then(function () { inFlight = false; });
  }

  function start() {
    if (timer) return;
    setLiveUI(true);
    poll();
    timer = window.setInterval(poll, POLL_MS);
  }

  function stop() {
    if (timer) {
      window.clearInterval(timer);
      timer = null;
    }
    setLiveUI(false);
  }

  toggle.addEventListener("click", function () {
    if (timer) stop();
    else start();
  });

  // Manual refresh: one in-place fetch. poll() guards overlap via inFlight, so it's click-safe.
  if (refreshBtn) {
    refreshBtn.addEventListener("click", function () { poll(); });
  }

  // Pause polling while the tab is hidden; resume if it was live.
  document.addEventListener("visibilitychange", function () {
    if (document.hidden) {
      if (timer) {
        window.clearInterval(timer);
        timer = null;
      }
    } else if (toggle.getAttribute("aria-pressed") === "true" && !timer) {
      poll();
      timer = window.setInterval(poll, POLL_MS);
    }
  });
})();
