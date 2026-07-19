// Radio playback + live now-playing metadata. Renders into every element carrying a
// js-np-* class (floating player pill, /live page) and binds every .js-radio-toggle
// button (play/pause). Turbo re-executes this body <script> tag on every in-site
// navigation, so setup is split into a one-time part (audio element listeners + the
// poll loop, guarded by window.__radioPlayerInit since #radio-audio lives inside the
// floating player's data-turbo-permanent wrapper and is never replaced) and a
// per-visit part that always runs (binding any freshly-rendered .js-radio-toggle,
// such as /live's own button, which Turbo destroys and recreates on every visit).
(function () {
  "use strict";

  var audio = document.getElementById("radio-audio");
  if (!audio) return;

  var STORAGE_KEY = "classyfm.playing";
  var POLL_MS = 15000;

  function applyState(toggle, state) {
    toggle.querySelectorAll("[data-state]").forEach(function (el) {
      el.classList.toggle("hidden", el.getAttribute("data-state") !== state);
    });
    var playing = state === "playing";
    toggle.setAttribute("aria-pressed", playing ? "true" : "false");
    toggle.setAttribute("aria-label", playing ? "Jeda radio" : "Putar radio");
  }

  function setState(state) {
    document.querySelectorAll(".js-radio-toggle").forEach(function (toggle) {
      applyState(toggle, state);
    });
  }

  function play() {
    setState("loading");
    // Reload the source so we always join the live edge, not a stale buffer.
    try { audio.load(); } catch (e) {}
    var p = audio.play();
    if (p && p.catch) {
      p.catch(function () {
        setState("paused");
        localStorage.removeItem(STORAGE_KEY);
      });
    }
  }

  function pause() {
    audio.pause();
    setState("paused");
  }

  function toggleClick() {
    if (audio.paused) {
      localStorage.setItem(STORAGE_KEY, "1");
      play();
    } else {
      localStorage.removeItem(STORAGE_KEY);
      pause();
    }
  }

  function renderNowPlaying(np) {
    var offline = !np.live;
    document.querySelectorAll(".js-np-title").forEach(function (el) {
      if (offline) {
        // Elements with a station name fallback (e.g. the floating player) show
        // the station name when offline instead of duplicating the "Offline"
        // live-status label shown elsewhere in the same widget.
        el.textContent = el.dataset.stationName || "Offline";
        return;
      }
      // Live: song info, else the on-air program's name, else the station name.
      el.textContent = np.has_song
        ? [np.artist, np.song].filter(Boolean).join(" - ")
        : (np.program_title || el.dataset.stationName || "");
    });

    document.querySelectorAll(".js-np-cover").forEach(function (el) {
      el.src = (!offline && np.cover_url) ? np.cover_url : "/static/img/default-cover.jpg";
    });

    document.querySelectorAll(".js-np-bitrate").forEach(function (el) {
      el.textContent = np.bitrate ? np.bitrate + " kbps" : "—";
    });
    document.querySelectorAll(".js-np-listeners").forEach(function (el) {
      el.textContent = typeof np.listeners === "number" ? np.listeners : "—";
    });
    document.querySelectorAll(".js-np-live-badge").forEach(function (el) {
      el.textContent = np.live ? "On Air" : "Offline";
      el.classList.toggle("text-green-600", np.live);
      el.classList.toggle("text-gray-400", !np.live);
    });
    document.querySelectorAll(".js-np-live-label").forEach(function (el) {
      el.textContent = np.live ? "Live" : "Offline";
    });

    document.querySelectorAll(".js-radio-toggle").forEach(function (toggle) {
      toggle.disabled = offline;
      toggle.classList.toggle("opacity-50", offline);
      toggle.classList.toggle("cursor-not-allowed", offline);
      toggle.setAttribute("aria-disabled", offline ? "true" : "false");
    });
    if (offline && !audio.paused) {
      audio.pause();
      setState("paused");
      localStorage.removeItem(STORAGE_KEY);
    }
  }

  function pollNowPlaying() {
    fetch("/api/nowplaying", { headers: { "Accept": "application/json" } })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (np) { if (np) renderNowPlaying(np); })
      .catch(function () {});
  }

  // Refresh now-playing data immediately on every visit (not just the first) so a
  // freshly-rendered /live page doesn't wait out the poll interval for real data.
  pollNowPlaying();

  var newButtons = [];
  document.querySelectorAll(".js-radio-toggle").forEach(function (toggle) {
    if (toggle.dataset.bound) return;
    toggle.dataset.bound = "1";
    toggle.addEventListener("click", toggleClick);
    newButtons.push(toggle);
  });

  if (window.__radioPlayerInit) {
    // Already set up on a prior visit - just sync any freshly-rendered button's
    // icon with the real (persisted) <audio> element's playback state.
    var state = audio.paused ? "paused" : "playing";
    newButtons.forEach(function (toggle) { applyState(toggle, state); });
    return;
  }
  window.__radioPlayerInit = true;

  audio.addEventListener("playing", function () { setState("playing"); });
  audio.addEventListener("pause", function () { setState("paused"); });
  audio.addEventListener("waiting", function () { setState("loading"); });
  audio.addEventListener("error", function () {
    setState("paused");
    localStorage.removeItem(STORAGE_KEY);
  });

  setInterval(pollNowPlaying, POLL_MS);

  // Resume playback if the user was listening before navigating.
  if (localStorage.getItem(STORAGE_KEY) === "1") {
    play();
  }
})();
