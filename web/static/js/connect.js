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

  function config() {
    var root = document.querySelector(".js-connect-root");
    return {
      isAdmin: !!root && root.dataset.connectAdmin === "1",
      csrf: root ? root.dataset.connectCsrf || "" : "",
    };
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
    var cfg = config();
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

    if (cfg.isAdmin) {
      var mod = document.createElement("div");
      mod.className = "mt-1 flex items-center gap-3";
      var del = document.createElement("button");
      del.type = "button";
      del.className = "js-connect-delete text-[11px] font-medium text-gray-400 transition-colors hover:text-signal";
      del.setAttribute("data-id", m.id);
      del.textContent = "Delete";
      mod.appendChild(del);
      var ban = document.createElement("button");
      ban.type = "button";
      ban.className = "js-connect-ban text-[11px] font-medium text-gray-400 transition-colors hover:text-signal";
      ban.setAttribute("data-user", m.user_id);
      ban.textContent = "Ban";
      mod.appendChild(ban);
      col.appendChild(mod);
    }

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

  function removeAll(id) {
    feeds().forEach(function (f) {
      var el = f.querySelector('[data-id="' + id + '"]');
      if (el) el.remove();
    });
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

  function poll() {
    var since = window.__connectLastId || 0;
    fetch(API + "?since=" + since, { headers: { "Accept": "application/json" } })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (data) {
        if (!data || !data.messages) return;
        data.messages.forEach(appendAll);
      })
      .catch(function () {});
  }

  function bindForm(form) {
    if (form.__cbound) return;
    form.__cbound = true;
    form.addEventListener("submit", function (e) {
      e.preventDefault();
      var input = form.querySelector(".js-connect-input");
      if (!input || !input.value.trim()) return;
      var payload = new URLSearchParams(new FormData(form)).toString();
      fetch("/connect/messages", {
        method: "POST",
        headers: { "Accept": "application/json", "Content-Type": "application/x-www-form-urlencoded" },
        body: payload,
      })
        .then(function (r) { return r.ok ? r.json() : null; })
        .then(function (data) {
          input.value = "";
          if (data && data.message) appendAll(data.message);
          else poll();
        })
        .catch(function () {});
    });
  }

  function moderate(url, csrf) {
    return fetch(url, {
      method: "POST",
      headers: { "Accept": "application/json", "Content-Type": "application/x-www-form-urlencoded" },
      body: "csrf_token=" + encodeURIComponent(csrf),
    }).then(function (r) { return r.ok; });
  }

  function bindFeed(feed) {
    if (feed.__cbound) return;
    feed.__cbound = true;
    feed.addEventListener("click", function (e) {
      var cfg = config();
      var del = e.target.closest(".js-connect-delete");
      if (del) {
        moderate("/connect/messages/" + del.getAttribute("data-id") + "/delete", cfg.csrf)
          .then(function (ok) { if (ok) removeAll(del.getAttribute("data-id")); });
        return;
      }
      var ban = e.target.closest(".js-connect-ban");
      if (ban) {
        if (!window.confirm("Ban this user from the chat?")) return;
        moderate("/connect/users/" + ban.getAttribute("data-user") + "/ban", cfg.csrf);
      }
    });
  }

  function bindWidget() {
    var toggle = document.querySelector(".js-connect-toggle");
    var panel = document.querySelector(".js-connect-panel");
    var close = document.querySelector(".js-connect-close");
    if (!toggle || !panel || toggle.__cbound) return;
    toggle.__cbound = true;
    function open(v) {
      panel.classList.toggle("hidden", !v);
      toggle.setAttribute("aria-expanded", v ? "true" : "false");
      if (v) {
        var f = panel.querySelector(".js-connect-feed");
        if (f) scrollToBottom(f);
      }
    }
    toggle.addEventListener("click", function () { open(panel.classList.contains("hidden")); });
    if (close) close.addEventListener("click", function () { open(false); });
  }

  // Run each execution (each Turbo navigation): hydrate any fresh feed, (re)bind
  // fresh forms/feeds, and bind the permanent widget once.
  feeds().forEach(function (f) { hydrate(f); bindFeed(f); });
  document.querySelectorAll(".js-connect-form").forEach(bindForm);
  bindWidget();

  if (window.__connectInit) return;
  window.__connectInit = true;
  setInterval(poll, POLL_MS);
})();
