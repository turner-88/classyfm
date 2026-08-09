// Drives the Connect chatroom on both surfaces: the site-wide floating widget
// (#connect-widget, a data-turbo-permanent element whose DOM + listeners survive Turbo
// navigations) and the dedicated /connect page (re-rendered on each visit). Both use the
// same .js-connect-* hooks and are hydrated from /api/connect/messages, so there is one
// rendering path for messages.
//
// Turbo re-executes this body <script> on every in-site navigation (see radio.js). The
// setInterval poll is registered once (window.__connectInit); per-element listeners are
// bound once each via an __cbound flag (the permanent widget must not double-bind, while
// a fresh page feed binds on arrival); the highest-seen message id lives on
// window.__connectLastId so it persists across navigations for the permanent widget.
(function () {
  "use strict";

  var POLL_MS = 5000;
  var API = "/api/connect/messages";

  if (!document.querySelector(".js-connect-feed")) {
    // No chat surface on this page (shouldn't happen - the widget is site-wide - but
    // guards against a page that omits the layout).
    return;
  }

  function feeds() {
    return Array.prototype.slice.call(document.querySelectorAll(".js-connect-feed"));
  }

  function atBottom(feed) {
    return feed.scrollHeight - feed.scrollTop - feed.clientHeight < 40;
  }

  function scrollToBottom(feed) {
    feed.scrollTop = feed.scrollHeight;
  }

  // buildMessage renders one message <li>. All user text goes in via textContent, never
  // innerHTML, so a message body can never inject markup (defense in depth on top of the
  // server-side sanitizer).
  function buildMessage(m) {
    var li = document.createElement("li");
    li.className = "js-connect-msg flex items-start gap-2.5";
    li.setAttribute("data-id", m.id);

    var av = document.createElement("div");
    av.className = "flex h-8 w-8 shrink-0 items-center justify-center overflow-hidden rounded-full bg-brand-700 text-xs font-bold text-white";
    if (m.avatar) {
      var img = document.createElement("img");
      img.src = m.avatar;
      img.alt = "";
      img.loading = "lazy";
      img.referrerPolicy = "no-referrer";
      img.className = "h-full w-full object-cover";
      av.appendChild(img);
    } else {
      av.textContent = (m.name || "?").charAt(0).toUpperCase();
    }
    li.appendChild(av);

    var col = document.createElement("div");
    col.className = "min-w-0 flex-1";

    var head = document.createElement("div");
    head.className = "flex items-center gap-1.5";
    var name = document.createElement("span");
    name.className = "truncate text-xs font-bold text-gray-900";
    name.textContent = m.name;
    head.appendChild(name);
    if (m.is_admin) {
      var badge = document.createElement("span");
      badge.className = "rounded-full bg-signal px-1.5 py-0.5 text-[10px] font-bold uppercase leading-none text-white";
      badge.textContent = "Admin";
      head.appendChild(badge);
    }
    var time = document.createElement("span");
    time.className = "ml-auto shrink-0 text-[10px] text-gray-400";
    time.textContent = m.time;
    head.appendChild(time);
    col.appendChild(head);

    var body = document.createElement("p");
    body.className = "mt-0.5 break-words text-sm text-gray-700";
    body.textContent = m.body;
    col.appendChild(body);

    li.appendChild(col);
    return li;
  }

  function noteLastId(id) {
    if (!window.__connectLastId || id > window.__connectLastId) window.__connectLastId = id;
  }

  // append adds a message to one feed unless it is already present (dedup by id), keeping
  // the view pinned to the bottom if the reader was already there.
  function append(feed, m) {
    if (feed.querySelector('[data-id="' + m.id + '"]')) return;
    var empty = feed.querySelector(".js-connect-empty");
    if (empty) empty.remove();
    var stick = atBottom(feed);
    feed.appendChild(buildMessage(m));
    if (stick) scrollToBottom(feed);
  }

  function appendAll(m) {
    noteLastId(m.id);
    feeds().forEach(function (f) { append(f, m); });
  }

  // hydrate fills a not-yet-populated feed with the recent history.
  function hydrate(feed) {
    if (feed.dataset.hydrated === "1") return;
    feed.dataset.hydrated = "1";
    fetch(API, { headers: { "Accept": "application/json" } })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (data) {
        if (!data || !data.messages) return;
        var empty = feed.querySelector(".js-connect-empty");
        if (data.messages.length === 0) {
          if (empty) empty.textContent = "No messages yet. Say hello!";
          return;
        }
        if (empty) empty.remove();
        data.messages.forEach(function (m) { append(feed, m); noteLastId(m.id); });
        scrollToBottom(feed);
      })
      .catch(function () { feed.dataset.hydrated = ""; });
  }

  // removeHidden deletes any on-screen message whose id a moderator has hidden. The server
  // ships these ids on every poll (independent of the since cursor, since a hidden message's
  // id is at or below it); removing an absent node is a harmless no-op, so this is idempotent.
  function removeHidden(ids) {
    if (!ids || !ids.length) return;
    feeds().forEach(function (feed) {
      ids.forEach(function (id) {
        var node = feed.querySelector('[data-id="' + id + '"]');
        if (node) node.remove();
      });
    });
  }

  function poll() {
    var since = window.__connectLastId || 0;
    fetch(API + "?since=" + since, { headers: { "Accept": "application/json" } })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (data) {
        if (!data) return;
        if (data.messages) data.messages.forEach(appendAll);
        removeHidden(data.hidden);
      })
      .catch(function () {});
  }

  // The composer shows a rejected post's reason inline (chat disabled, banned, etc.). The
  // .js-connect-error slot lives inside the same .js-connect-composer as the form, so each
  // composer instance (widget + /connect page) reports into its own alert.
  function errorEl(form) {
    var c = form.closest(".js-connect-composer");
    return c ? c.querySelector(".js-connect-error") : null;
  }

  function showError(form, msg) {
    var el = errorEl(form);
    if (!el) return;
    el.textContent = msg;
    el.classList.remove("hidden");
  }

  function clearError(form) {
    var el = errorEl(form);
    if (!el) return;
    el.textContent = "";
    el.classList.add("hidden");
  }

  function bindForm(form) {
    if (form.__cbound) return;
    form.__cbound = true;
    form.addEventListener("submit", function (e) {
      e.preventDefault();
      var input = form.querySelector(".js-connect-input");
      if (!input || !input.value.trim()) return;
      clearError(form);
      var payload = new URLSearchParams(new FormData(form)).toString();
      fetch("/connect/messages", {
        method: "POST",
        headers: { "Accept": "application/json", "Content-Type": "application/x-www-form-urlencoded" },
        body: payload,
      })
        .then(function (r) {
          // The error paths answer plain text (see connectRespond); surface the server's
          // reason and keep the typed text so the user can retry.
          if (!r.ok) {
            return r.text().then(function (t) {
              showError(form, (t && t.trim()) || "Gagal mengirim pesan. Coba lagi.");
              return null;
            });
          }
          return r.json();
        })
        .then(function (data) {
          if (data === null) return;
          input.value = "";
          if (data && data.message) appendAll(data.message);
          else poll();
        })
        .catch(function () { showError(form, "Gagal mengirim pesan. Coba lagi."); });
    });
  }

  // bindLogout intercepts the "Sign out" form so the composer updates in place instead
  // of a Turbo visit (which would keep the stale signed-in composer inside the permanent
  // #floating-stack) or a full reload. The server returns the freshly-rendered signed-out
  // composer, which we swap into every composer container (widget + /connect page). On any
  // failure we fall back to a hard reload, which the server also renders correctly.
  function bindLogout(form) {
    if (form.__cbound) return;
    form.__cbound = true;
    form.addEventListener("submit", function (e) {
      e.preventDefault();
      var payload = new URLSearchParams(new FormData(form)).toString();
      fetch("/connect/logout", {
        method: "POST",
        headers: {
          "Accept": "text/html",
          "X-Requested-With": "fetch",
          "Content-Type": "application/x-www-form-urlencoded",
        },
        body: payload,
      })
        .then(function (r) { return r.ok ? r.text() : null; })
        .then(function (html) {
          if (html == null) { window.location.reload(); return; }
          document.querySelectorAll(".js-connect-composer").forEach(function (c) {
            c.innerHTML = html;
          });
        })
        .catch(function () { window.location.reload(); });
    });
  }

  function bindWidget() {
    var toggle = document.querySelector(".js-connect-toggle");
    var panel = document.querySelector(".js-connect-panel");
    var close = document.querySelector(".js-connect-close");
    if (!toggle || !panel || toggle.__cbound) return;
    toggle.__cbound = true;
    // Persisted per hard reload as "classyfm.connect.open" ("1"/"0"); across in-site
    // Turbo navigations the permanent node carries its state and this bind is skipped.
    function open(v) {
      panel.classList.toggle("hidden", !v);
      toggle.classList.toggle("hidden", v);
      toggle.setAttribute("aria-expanded", v ? "true" : "false");
      try { localStorage.setItem("classyfm.connect.open", v ? "1" : "0"); } catch (e) {}
      if (v) {
        var f = panel.querySelector(".js-connect-feed");
        if (f) scrollToBottom(f);
      }
    }
    toggle.addEventListener("click", function () { open(panel.classList.contains("hidden")); });
    if (close) close.addEventListener("click", function () { open(false); });

    // Restore the prior choice on a fresh load. The panel renders open by default, so an
    // absent value leaves it as-is; only a stored "0" needs to collapse it (and a stored
    // "1" re-asserts open, scrolling the just-hydrated feed to the bottom).
    var stored = null;
    try { stored = localStorage.getItem("classyfm.connect.open"); } catch (e) {}
    if (stored === "0") open(false);
    else if (stored === "1") open(true);
  }

  // Run each execution (each Turbo navigation): hydrate any fresh feed, (re)bind
  // fresh forms, and bind the permanent widget once.
  feeds().forEach(function (f) { hydrate(f); });
  document.querySelectorAll(".js-connect-form").forEach(bindForm);
  document.querySelectorAll(".js-connect-logout").forEach(bindLogout);
  bindWidget();

  if (window.__connectInit) return;
  window.__connectInit = true;
  setInterval(poll, POLL_MS);
})();
