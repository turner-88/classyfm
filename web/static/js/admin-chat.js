// Admin Connect-chat moderation: a live, newest-first list of the recent chat messages
// with a per-message delete. Like the public Connect widget (connect.js), this runs
// entirely client-side against the mobile app's Firebase Realtime Database - there is no
// server-side chat store - so it signs in with Google separately from the admin session
// and reads/writes RTDB directly. The window.fb* singletons come from firebase-config.js
// (loaded just before this on admin/chat.html).
//
// Deleting = db.ref(chatsPath + "/" + key).remove(). Whether that succeeds is governed by
// the RTDB security rules (coordinated with the app dev), not by this page; a rejected
// delete surfaces its Firebase error code (e.g. permission_denied) in the error banner.
// The public widget and this page both listen on child_removed, so a successful delete
// disappears everywhere live.
//
// Admin pages are full page loads (no Turbo), so this runs once per load - no permanence
// guards needed.
(function () {
  "use strict";

  var root = document.querySelector("[data-adminchat]");
  if (!root) return;

  var db = window.fbDatabase;
  var auth = window.fbAuth;
  var chatsPath = window.fbChatsPath;

  function q(sel) { return root.querySelector(sel); }

  var unavailableEl = q(".js-adminchat-unavailable");
  var errorEl = q(".js-adminchat-error");
  var signedOutEl = q(".js-adminchat-signedout");
  var signedInEl = q(".js-adminchat-signedin");
  var identityEl = q(".js-adminchat-identity");
  var emailEl = q(".js-adminchat-email");
  var avatarEl = q(".js-adminchat-avatar");
  var feedEl = q(".js-adminchat-feed");

  function show(el) { if (el) el.classList.remove("hidden"); }
  function hide(el) { if (el) el.classList.add("hidden"); }

  function showError(msg) {
    if (!errorEl) return;
    errorEl.textContent = msg;
    errorEl.classList.remove("hidden");
  }
  function clearError() {
    if (!errorEl) return;
    errorEl.textContent = "";
    errorEl.classList.add("hidden");
  }

  // Firebase not configured/loaded (see firebase-config.js): degrade gracefully.
  if (!db || !auth || !chatsPath) {
    hide(signedOutEl);
    hide(signedInEl);
    show(unavailableEl);
    return;
  }

  // Clock time in the station's timezone (WIB), matching the public feed.
  var timeFmt = new Intl.DateTimeFormat("en-GB", {
    timeZone: "Asia/Jakarta", year: "numeric", month: "short", day: "2-digit",
    hour: "2-digit", minute: "2-digit", hour12: false,
  });
  function fmtTime(ms) {
    if (!ms || typeof ms !== "number") return "";
    try { return timeFmt.format(new Date(ms)); } catch (e) { return ""; }
  }

  // fillAvatar renders a photo (referrer-stripped, lazy) into a pre-styled circle, or falls
  // back to the first letter of the label. Shared by the signed-in card and message rows.
  function fillAvatar(el, imageUrl, label) {
    if (!el) return;
    el.textContent = "";
    if (imageUrl) {
      var img = document.createElement("img");
      img.src = imageUrl;
      img.alt = "";
      img.loading = "lazy";
      img.referrerPolicy = "no-referrer";
      img.className = "h-full w-full object-cover";
      el.appendChild(img);
    } else {
      el.textContent = (label || "?").charAt(0).toUpperCase();
    }
  }

  // Admin badge set: the app's /classiers list keyed by uid, exactly as connect.js
  // derives it (there is no is_admin flag on messages). Loaded once per sign-in.
  var admins = {};
  function loadAdmins() {
    return db.ref("/classiers").once("value").then(function (snap) {
      admins = {};
      snap.forEach(function (c) {
        var v = c.val();
        if (v && v.uid) admins[v.uid] = true;
      });
    }).catch(function () { /* no badges rather than a broken feed */ });
  }
  function isAdminUid(uid) { return !!(uid && admins[uid]); }

  // ---- feed rendering -------------------------------------------------------

  function emptyEl() { return feedEl.querySelector(".js-adminchat-empty"); }
  function ensureEmpty(text) {
    if (feedEl.querySelector(".js-adminchat-msg")) return;
    var e = emptyEl();
    if (!e) {
      e = document.createElement("li");
      e.className = "js-adminchat-empty px-4 py-8 text-center text-sm text-gray-400";
      feedEl.appendChild(e);
    }
    e.textContent = text;
  }

  // buildRow renders one message. All user-supplied text goes in via textContent, never
  // innerHTML, so a message body can never inject markup.
  function buildRow(m) {
    var li = document.createElement("li");
    li.className = "js-adminchat-msg flex items-start gap-3 px-4 py-3";
    li.setAttribute("data-id", m.key);

    var av = document.createElement("div");
    av.className = "flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-full bg-brand-700 text-sm font-bold text-white";
    fillAvatar(av, m.image, m.author);
    li.appendChild(av);

    var col = document.createElement("div");
    col.className = "min-w-0 flex-1";

    var head = document.createElement("div");
    head.className = "flex flex-wrap items-center gap-2";
    var name = document.createElement("span");
    name.className = "truncate text-sm font-bold text-gray-900";
    name.textContent = m.author || "(no name)";
    head.appendChild(name);
    if (isAdminUid(m.uid)) {
      var badge = document.createElement("span");
      badge.className = "rounded-full bg-signal px-1.5 py-0.5 text-[10px] font-bold uppercase leading-none text-white";
      badge.textContent = "Admin";
      head.appendChild(badge);
    }
    var time = document.createElement("span");
    time.className = "text-xs text-gray-400";
    time.textContent = fmtTime(m.time);
    head.appendChild(time);
    col.appendChild(head);

    // Quoted reply, if any (matches the shape connect.js/the app write).
    if (m.reply && (m.reply.author || m.reply.body)) {
      var quote = document.createElement("div");
      quote.className = "mt-1 border-l-2 border-gray-200 pl-2 text-xs text-gray-500";
      var qa = document.createElement("span");
      qa.className = "font-semibold";
      qa.textContent = (m.reply.author || "") + ": ";
      quote.appendChild(qa);
      quote.appendChild(document.createTextNode(m.reply.body || ""));
      col.appendChild(quote);
    }

    var body = document.createElement("p");
    body.className = "mt-0.5 break-words text-sm text-gray-800";
    body.textContent = m.body || "";
    col.appendChild(body);

    var uid = document.createElement("p");
    uid.className = "mt-0.5 truncate font-mono text-[10px] text-gray-400";
    uid.textContent = "uid: " + (m.uid || "(unknown)");
    col.appendChild(uid);

    li.appendChild(col);

    var del = document.createElement("button");
    del.type = "button";
    del.className = "icon-btn-danger shrink-0";
    del.title = "Delete message";
    del.setAttribute("aria-label", "Delete message");
    var delIcon = document.createElement("span");
    delIcon.className = "block h-4 w-4";
    // Reuse the same trash glyph the rest of the admin UI uses.
    delIcon.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" class="h-full w-full"><path d="M10 11v6"/><path d="M14 11v6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/><path d="M3 6h18"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>';
    del.appendChild(delIcon);
    del.addEventListener("click", function () {
      var who = m.author ? (" from " + m.author) : "";
      if (!window.confirm("Delete this message" + who + "? This removes it for everyone.")) return;
      clearError();
      db.ref(chatsPath + "/" + m.key).remove().catch(function (err) {
        var code = (err && err.code) || (err && err.message) || "unknown";
        showError("Failed to delete message (" + code + ").");
      });
    });
    li.appendChild(del);

    return li;
  }

  // Newest-first: child_added fires oldest->newest for the initial window and for each new
  // message, so prepending always lands the most recent at the top.
  function addMessage(m) {
    if (!m.key) return;
    if (feedEl.querySelector('[data-id="' + m.key + '"]')) return;
    var e = emptyEl();
    if (e) e.remove();
    feedEl.insertBefore(buildRow(m), feedEl.firstChild);
  }
  function removeMessage(key) {
    var node = feedEl.querySelector('[data-id="' + key + '"]');
    if (node) node.remove();
    ensureEmpty("No messages.");
  }

  // ---- live wiring (auth-gated) --------------------------------------------

  var feedOn = false;
  var query = null;
  function attachFeed() {
    if (feedOn) return;
    feedOn = true;
    query = db.ref(chatsPath).limitToLast(200);
    query.on("child_added", function (snap) {
      var m = snap.val() || {};
      if (!m.key) m.key = snap.key;
      addMessage(m);
    }, function (err) {
      // Read cancelled/denied (e.g. rules deny read to this account).
      var code = (err && err.code) || (err && err.message) || "unknown";
      showError("Cannot read chat (" + code + ").");
    });
    query.on("child_removed", function (snap) { removeMessage(snap.key); });
  }
  function detachFeed() {
    if (query) { query.off(); query = null; }
    feedOn = false;
  }
  function clearFeed() {
    feedEl.querySelectorAll(".js-adminchat-msg").forEach(function (n) { n.remove(); });
    ensureEmpty("Loading messages…");
  }

  // ---- sign-in / sign-out ---------------------------------------------------

  function bindClick(sel, fn) {
    var el = q(sel);
    if (el) el.addEventListener("click", fn);
  }

  bindClick(".js-adminchat-signin", function () {
    clearError();
    var provider = new firebase.auth.GoogleAuthProvider();
    auth.signInWithPopup(provider).catch(function (err) {
      var code = (err && err.code) || "";
      if (code === "auth/popup-closed-by-user" || code === "auth/cancelled-popup-request") return;
      showError("Sign-in failed (" + (code || "unknown") + ").");
    });
  });

  bindClick(".js-adminchat-signout", function () { auth.signOut(); });

  auth.onAuthStateChanged(function (user) {
    clearError();
    if (user) {
      hide(signedOutEl);
      show(signedInEl);
      // Identity card: avatar + name (bold) over email (muted). If the account has no
      // display name the email becomes the primary line and the second line is left empty.
      var name = user.displayName || "";
      var email = user.email || "";
      if (identityEl) identityEl.textContent = name || email;
      if (emailEl) emailEl.textContent = name ? email : "";
      fillAvatar(avatarEl, user.photoURL || "", name || email);
      clearFeed();
      loadAdmins().then(attachFeed);
    } else {
      detachFeed();
      admins = {};
      clearFeed();
      hide(signedInEl);
      show(signedOutEl);
    }
  });
})();
