// Drives the Connect chatroom on both surfaces: the site-wide floating widget
// (#connect-widget, a data-turbo-permanent element whose DOM + listeners survive Turbo
// navigations) and the dedicated /connect page (re-rendered on each visit). Both use the
// same .js-connect-* hooks.
//
// The chat is backed by the mobile app's Firebase Realtime Database (see firebase-config.js),
// so the website and the app share one live room with real-time push (no polling). Messages
// live at window.fbChatsPath (/chats in prod, /chatsdev in dev). Auth is Firebase
// Authentication (Google); the composer's signed-in / signed-out state is driven client-side
// from onAuthStateChanged.
//
// Turbo re-executes this body <script> on every in-site navigation (see radio.js). The RTDB
// listeners and the auth observer are registered once (window.__connectInit); per-element
// listeners are bound once each via an __cbound flag (the permanent widget must not
// double-bind, while a fresh page feed binds on arrival).
(function () {
  "use strict";

  if (!document.querySelector(".js-connect-feed")) {
    // No chat surface on this page (shouldn't happen - the widget is site-wide - but
    // guards against a page that omits the layout).
    return;
  }

  var db = window.fbDatabase;
  var auth = window.fbAuth;
  var chatsPath = window.fbChatsPath;
  var MAX_BODY = 1000;

  // Format an epoch-ms timestamp as clock time in the station's timezone (WIB) - stable
  // (no client ticking) and unambiguous for a live chat, matching the old server render.
  var timeFmt = new Intl.DateTimeFormat("en-GB", {
    timeZone: "Asia/Jakarta", hour: "2-digit", minute: "2-digit", hour12: false,
  });
  function fmtTime(ms) {
    if (!ms || typeof ms !== "number") return "";
    try { return timeFmt.format(new Date(ms)); } catch (e) { return ""; }
  }

  function feeds() {
    return Array.prototype.slice.call(document.querySelectorAll(".js-connect-feed"));
  }
  function composers() {
    return Array.prototype.slice.call(document.querySelectorAll(".js-connect-composer"));
  }
  function atBottom(feed) {
    return feed.scrollHeight - feed.scrollTop - feed.clientHeight < 40;
  }
  function scrollToBottom(feed) {
    feed.scrollTop = feed.scrollHeight;
  }

  // isAdminUid reports whether a message author carries the "special" badge. The badge set
  // is the app's /classiers list (loaded once into window.__connectAdmins), matched by uid -
  // there is no is_admin flag on messages, so both clients derive it the same way.
  function isAdminUid(uid) {
    return !!(uid && window.__connectAdmins && window.__connectAdmins[uid]);
  }

  // buildMessage renders one message <li>. All user text goes in via textContent, never
  // innerHTML, so a message body can never inject markup (defense in depth: RTDB security
  // rules are the server-side guard).
  function buildMessage(m) {
    var li = document.createElement("li");
    li.className = "js-connect-msg flex items-start gap-2.5";
    li.setAttribute("data-id", m.key);

    var av = document.createElement("div");
    av.className = "flex h-8 w-8 shrink-0 items-center justify-center overflow-hidden rounded-full bg-brand-700 text-xs font-bold text-white";
    if (m.image) {
      var img = document.createElement("img");
      img.src = m.image;
      img.alt = "";
      img.loading = "lazy";
      img.referrerPolicy = "no-referrer";
      img.className = "h-full w-full object-cover";
      av.appendChild(img);
    } else {
      av.textContent = (m.author || "?").charAt(0).toUpperCase();
    }
    li.appendChild(av);

    var col = document.createElement("div");
    col.className = "min-w-0 flex-1";

    var head = document.createElement("div");
    head.className = "flex items-center gap-1.5";
    var name = document.createElement("span");
    name.className = "truncate text-xs font-bold text-gray-900";
    name.textContent = m.author || "";
    head.appendChild(name);
    if (isAdminUid(m.uid)) {
      var badge = document.createElement("span");
      badge.className = "rounded-full bg-signal px-1.5 py-0.5 text-[10px] font-bold uppercase leading-none text-white";
      badge.textContent = "Admin";
      head.appendChild(badge);
    }
    var time = document.createElement("span");
    time.className = "ml-auto shrink-0 text-[10px] text-gray-400";
    time.textContent = fmtTime(m.time);
    head.appendChild(time);
    col.appendChild(head);

    // A quoted reply (app-authored feature): render the quoted line, then the body. The web
    // has no reply composer of its own, but rendering incoming replies keeps parity.
    if (m.reply && (m.reply.author || m.reply.body)) {
      var quote = document.createElement("div");
      quote.className = "mt-0.5 border-l-2 border-gray-300 pl-2 text-xs text-gray-400";
      var qa = document.createElement("span");
      qa.className = "font-semibold";
      qa.textContent = (m.reply.author || "") + ": ";
      quote.appendChild(qa);
      quote.appendChild(document.createTextNode(m.reply.body || ""));
      col.appendChild(quote);
    }

    var body = document.createElement("p");
    body.className = "mt-0.5 break-words text-sm text-gray-700";
    body.textContent = m.body || "";
    col.appendChild(body);

    li.appendChild(col);
    return li;
  }

  // append adds a message to one feed unless it is already present (dedup by key), keeping
  // the view pinned to the bottom if the reader was already there.
  function append(feed, m) {
    if (!m.key) return;
    if (feed.querySelector('[data-id="' + m.key + '"]')) return;
    var empty = feed.querySelector(".js-connect-empty");
    if (empty) empty.remove();
    var stick = atBottom(feed);
    feed.appendChild(buildMessage(m));
    if (stick) scrollToBottom(feed);
  }
  function appendAll(m) {
    feeds().forEach(function (f) { append(f, m); });
  }
  // removeKey deletes an on-screen message across all feeds (moderator hard-delete, or an old
  // message sliding out of the limitToLast window). Removing an absent node is a no-op.
  function removeKey(key) {
    feeds().forEach(function (feed) {
      var node = feed.querySelector('[data-id="' + key + '"]');
      if (node) node.remove();
    });
  }

  // adminReady loads the /classiers badge set once, shared across navigations. hydrate waits
  // on it so the first render already has correct badges.
  function loadAdmins() {
    if (window.__connectAdminReady) return window.__connectAdminReady;
    window.__connectAdmins = {};
    window.__connectAdminReady = db.ref("/classiers").once("value")
      .then(function (snap) {
        snap.forEach(function (c) {
          var v = c.val();
          if (v && v.uid) window.__connectAdmins[v.uid] = true;
        });
      })
      .catch(function () { /* no badges rather than a broken feed */ });
    return window.__connectAdminReady;
  }

  // hydrate fills a not-yet-populated feed with the recent history (last 50). The global
  // child_added listener (registered once below) keeps every feed live thereafter.
  function hydrate(feed) {
    if (feed.dataset.hydrated === "1") return;
    feed.dataset.hydrated = "1";
    loadAdmins().then(function () {
      return db.ref(chatsPath).limitToLast(50).once("value");
    }).then(function (snap) {
      var empty = feed.querySelector(".js-connect-empty");
      if (!snap || !snap.exists() || snap.numChildren() === 0) {
        if (empty) empty.textContent = "No messages yet. Say hello!";
        return;
      }
      if (empty) empty.remove();
      snap.forEach(function (c) {
        var m = c.val() || {};
        if (!m.key) m.key = c.key;
        append(feed, m);
      });
      scrollToBottom(feed);
    }).catch(function () { feed.dataset.hydrated = ""; });
  }

  // ---- composer: inline error + auth-driven state --------------------------

  function errorEl(composer) { return composer.querySelector(".js-connect-error"); }
  function showError(el, msg) {
    if (!el) return;
    el.textContent = msg;
    el.classList.remove("hidden");
  }
  function clearError(el) {
    if (!el) return;
    el.textContent = "";
    el.classList.add("hidden");
  }

  // ---- live feed wiring (auth-gated) ---------------------------------------
  //
  // The database rules require an authenticated user even to READ, so the feed listeners are
  // attached only once a user is signed in, and torn down on sign-out. child_added fires for
  // the initial last-50 window and every new message; child_removed for a hard-delete or an
  // item sliding out of that window. Both fan out to every feed on the page.

  function attachFeed() {
    if (window.__connectFeedOn) return;
    window.__connectFeedOn = true;
    var query = db.ref(chatsPath).limitToLast(50);
    window.__connectQuery = query;
    query.on("child_added", function (snap) {
      var m = snap.val() || {};
      if (!m.key) m.key = snap.key;
      appendAll(m);
    }, function () { /* read cancelled (e.g. after sign-out) - detachFeed handles cleanup */ });
    query.on("child_removed", function (snap) { removeKey(snap.key); });
  }

  function detachFeed() {
    if (window.__connectQuery) {
      window.__connectQuery.off();
      window.__connectQuery = null;
    }
    window.__connectFeedOn = false;
  }

  // resetFeeds clears rendered messages and shows a prompt (used on sign-out, since a
  // signed-out visitor cannot read the chat under the database rules).
  function resetFeeds(promptText) {
    feeds().forEach(function (feed) {
      feed.querySelectorAll(".js-connect-msg").forEach(function (n) { n.remove(); });
      feed.dataset.hydrated = "";
      var empty = feed.querySelector(".js-connect-empty");
      if (!empty) {
        empty = document.createElement("li");
        empty.className = "js-connect-empty m-auto text-center text-sm text-gray-400";
        feed.appendChild(empty);
      }
      empty.textContent = promptText;
    });
  }

  // applyAuthState reflects the current auth state into every composer AND the feed. Called on
  // each execution (so freshly navigated surfaces sync) and on every auth change. Reads and
  // writes both require auth, so the feed is shown only when signed in.
  function applyAuthState(user) {
    composers().forEach(function (c) {
      var signedIn = c.querySelector(".js-connect-signedin");
      var signedOut = c.querySelector(".js-connect-signedout");
      var unavailable = c.querySelector(".js-connect-unavailable");
      if (unavailable) unavailable.classList.add("hidden");
      if (signedIn) signedIn.classList.toggle("hidden", !user);
      if (signedOut) signedOut.classList.toggle("hidden", !!user);
      if (user) {
        var uname = c.querySelector(".js-connect-username");
        if (uname) {
          uname.textContent = "Signed in as " + (user.displayName || user.email || "") +
            (isAdminUid(user.uid) ? " · Admin" : "");
        }
      }
    });

    if (user) {
      attachFeed();
      feeds().forEach(hydrate);
    } else {
      detachFeed();
      // Force a re-read of the badge list (needs auth) after the next sign-in.
      window.__connectAdminReady = null;
      window.__connectAdmins = {};
      resetFeeds("Sign in with Google to view the live chat.");
    }
  }

  // markUnavailable is the fallback when Firebase isn't configured/loaded: hide both
  // interactive branches and show the "unavailable" note.
  function markUnavailable() {
    composers().forEach(function (c) {
      ["js-connect-signedin", "js-connect-signedout"].forEach(function (cls) {
        var el = c.querySelector("." + cls);
        if (el) el.classList.add("hidden");
      });
      var un = c.querySelector(".js-connect-unavailable");
      if (un) un.classList.remove("hidden");
    });
    feeds().forEach(function (f) {
      var e = f.querySelector(".js-connect-empty");
      if (e) e.textContent = "Chat is temporarily unavailable.";
    });
  }

  // sendMessage writes to RTDB in the app's message shape ({author, body, image, key, time,
  // uid}), setting key to the push id so both clients read it identically.
  function sendMessage(text, errEl) {
    var user = auth.currentUser;
    if (!user) { showError(errEl, "Silakan masuk dengan Google untuk mengirim pesan."); return; }
    var ref = db.ref(chatsPath).push();
    ref.set({
      author: user.displayName || user.email || "Anonymous",
      body: text,
      image: user.photoURL || "",
      key: ref.key,
      time: firebase.database.ServerValue.TIMESTAMP,
      uid: user.uid,
    }).catch(function () {
      showError(errEl, "Gagal mengirim pesan. Coba lagi.");
    });
  }

  function bindForm(form) {
    if (form.__cbound) return;
    form.__cbound = true;
    var composer = form.closest(".js-connect-composer");
    form.addEventListener("submit", function (e) {
      e.preventDefault();
      var input = form.querySelector(".js-connect-input");
      if (!input) return;
      var text = input.value.trim();
      if (!text) return;
      if (text.length > MAX_BODY) text = text.slice(0, MAX_BODY);
      clearError(errorEl(composer));
      input.value = "";
      sendMessage(text, errorEl(composer));
    });
  }

  function bindSignIn(btn) {
    if (btn.__cbound) return;
    btn.__cbound = true;
    btn.addEventListener("click", function () {
      var provider = new firebase.auth.GoogleAuthProvider();
      auth.signInWithPopup(provider).then(function (res) {
        // Upsert the profile at /users/{uid} to match the app's shape.
        var u = res && res.user;
        if (u) {
          db.ref("/users/" + u.uid).update({
            email: u.email || "",
            image: u.photoURL || "",
            name: u.displayName || "",
          }).catch(function () {});
        }
      }).catch(function (err) {
        // Popup blocked or dismissed - surface it in whichever composer holds this button.
        var composer = btn.closest(".js-connect-composer");
        if (err && err.code !== "auth/popup-closed-by-user" && err.code !== "auth/cancelled-popup-request") {
          showError(errorEl(composer), "Gagal masuk. Coba lagi.");
        }
      });
    });
  }

  function bindSignOut(btn) {
    if (btn.__cbound) return;
    btn.__cbound = true;
    btn.addEventListener("click", function () { auth.signOut(); });
  }

  function bindWidget() {
    var toggle = document.querySelector(".js-connect-toggle");
    var panel = document.querySelector(".js-connect-panel");
    var close = document.querySelector(".js-connect-close");
    if (!toggle || !panel || toggle.__cbound) return;
    toggle.__cbound = true;
    // Persisted per hard reload as "classyfm.connect.open" ("1"/"0"); across in-site Turbo
    // navigations the permanent node carries its state and this bind is skipped. persist
    // defaults to true; the narrow-screen default passes persist=false so a responsive
    // collapse never becomes a sticky preference - mirroring floating-widgets.js.
    function open(v, persist) {
      panel.classList.toggle("hidden", !v);
      toggle.classList.toggle("hidden", v);
      toggle.setAttribute("aria-expanded", v ? "true" : "false");
      if (persist !== false) {
        try { localStorage.setItem("classyfm.connect.open", v ? "1" : "0"); } catch (e) {}
      }
      if (v) {
        var f = panel.querySelector(".js-connect-feed");
        if (f) scrollToBottom(f);
      }
    }
    toggle.addEventListener("click", function () { open(panel.classList.contains("hidden")); });
    if (close) close.addEventListener("click", function () { open(false); });

    var stored = null;
    try { stored = localStorage.getItem("classyfm.connect.open"); } catch (e) {}
    if (stored === "0") open(false);
    else if (stored === "1") open(true);
    else if (window.matchMedia("(max-width: 639px)").matches) open(false, false);
  }

  // ---- run ------------------------------------------------------------------

  bindWidget();

  if (!db || !auth || !chatsPath) {
    // Firebase not configured/loaded (see firebase-config.js). Degrade gracefully.
    markUnavailable();
    return;
  }

  // Per execution (each Turbo navigation): reflect the current auth state into any fresh
  // surface (which attaches/hydrates the feed when signed in) and (re)bind fresh controls.
  applyAuthState(auth.currentUser);
  document.querySelectorAll(".js-connect-form").forEach(bindForm);
  document.querySelectorAll(".js-connect-signin").forEach(bindSignIn);
  document.querySelectorAll(".js-connect-logout").forEach(bindSignOut);

  if (window.__connectInit) return;
  window.__connectInit = true;

  // The auth observer is the single driver of the feed: it attaches the RTDB listeners on
  // sign-in and tears them down on sign-out (reads require auth).
  auth.onAuthStateChanged(function (user) { applyAuthState(user); });
})();
