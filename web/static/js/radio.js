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

  // The floating player is data-turbo-permanent, so Turbo carries the pre-navigation
  // DOM node into every new page as-is rather than using whatever the server rendered
  // for it there - the server-side "hidden on /live" class on the expand link never
  // takes effect once Turbo is driving navigation. Re-derive it from the current URL
  // every visit: the link points at /live, so it only makes sense off that page.
  document.querySelectorAll(".js-widget-expand").forEach(function (el) {
    el.classList.toggle("hidden", location.pathname === "/live");
  });

  // Mirrors playback state into any element carrying the three [data-state] spans.
  // .js-radio-state elements (the collapsed player's bubble) get the icon swap but
  // not the aria-pressed/label pair: they aren't play/pause controls, they only
  // bring the collapsed card back, and promising "Pause radio" there would lie.
  function applyState(el, state) {
    el.querySelectorAll("[data-state]").forEach(function (s) {
      s.classList.toggle("hidden", s.getAttribute("data-state") !== state);
    });
    if (!el.classList.contains("js-radio-toggle")) return;
    var playing = state === "playing";
    el.setAttribute("aria-pressed", playing ? "true" : "false");
    el.setAttribute("aria-label", playing ? "Pause radio" : "Play radio");
  }

  function setState(state) {
    document.querySelectorAll(".js-radio-toggle, .js-radio-state").forEach(function (el) {
      applyState(el, state);
    });
    // One class on <body> drives every playing-only affordance (equalizer bars,
    // the play button's ripple arcs) via CSS, so widgets don't each need a hook.
    document.body.classList.toggle("is-playing", state === "playing");
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

  // firstText returns the first non-empty entry of list, else fallback.
  function firstText(list, fallback) {
    for (var i = 0; i < list.length; i++) {
      if (list[i]) return list[i];
    }
    return fallback || "";
  }

  function renderNowPlaying(np) {
    var offline = !np.live;

    // The floating card's and /live's two text lines. Each resolves its own
    // fallback chain - song/artist, then the on-air program's name/time range,
    // then the station's name/slogan (server-rendered into data attributes) -
    // rather than both switching tiers together, so a song with no artist tag
    // still shows the program's time range instead of a blank line. Offline
    // skips straight to the station tier, since the "Offline" state is already
    // spelled out by the live badge/label in the same widget.
    var timeRange = [np.program_start, np.program_end].filter(Boolean).join(" - ");
    var titles = offline ? [] : [np.has_song ? np.song : "", np.program_title];
    var subtitles = offline ? [] : [np.has_song ? np.artist : "", timeRange];
    document.querySelectorAll(".js-np-song").forEach(function (el) {
      el.textContent = firstText(titles, el.dataset.stationName);
    });
    document.querySelectorAll(".js-np-artist").forEach(function (el) {
      el.textContent = firstText(subtitles, el.dataset.stationSlogan);
    });

    // /live's announcer badge: hidden outright when nothing is on air (or the
    // program has no host), since an "Announcer:" label with no name is noise.
    var announcer = offline ? "" : (np.program_host || "");
    document.querySelectorAll(".js-np-announcer").forEach(function (el) {
      el.classList.toggle("hidden", !announcer);
    });
    document.querySelectorAll(".js-np-announcer-name").forEach(function (el) {
      el.textContent = announcer;
    });

    document.querySelectorAll(".js-np-cover").forEach(function (el) {
      el.src = (!offline && np.cover_url) ? np.cover_url : "/static/img/default-cover.jpg";
    });

    // The badge's own text lives in .js-np-live-status - the badge element also
    // holds a dot that inherits its (signal-red/gray) colour, so it can't be
    // rewritten wholesale here.
    document.querySelectorAll(".js-np-live-badge").forEach(function (el) {
      el.classList.toggle("text-signal", np.live);
      el.classList.toggle("text-gray-400", !np.live);
    });
    document.querySelectorAll(".js-np-live-status").forEach(function (el) {
      el.textContent = np.live ? "On Air" : "Offline";
    });
    document.querySelectorAll(".js-np-live-label").forEach(function (el) {
      el.textContent = np.live ? "Now Playing" : "Offline";
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

  var fresh = [];
  document.querySelectorAll(".js-radio-toggle").forEach(function (toggle) {
    if (toggle.dataset.bound) return;
    toggle.dataset.bound = "1";
    toggle.addEventListener("click", toggleClick);
    fresh.push(toggle);
  });

  // State-only mirrors: no click handler here, and deliberately never disabled by
  // renderNowPlaying either - they just reveal a collapsed card, which has to stay
  // possible while the stream is off air. Guarded by its own attribute rather than
  // data-bound, which means "has a listener".
  document.querySelectorAll(".js-radio-state").forEach(function (el) {
    if (el.dataset.stateSynced) return;
    el.dataset.stateSynced = "1";
    fresh.push(el);
  });

  // Nav links to /live: start playback (never pause) on click, then let the
  // anchor's normal navigation proceed - unlike .js-radio-toggle this is never a
  // toggle, since pausing on a nav click would be surprising.
  document.querySelectorAll(".js-radio-play-link").forEach(function (link) {
    if (link.dataset.bound) return;
    link.dataset.bound = "1";
    link.addEventListener("click", function () {
      if (audio.paused) {
        localStorage.setItem(STORAGE_KEY, "1");
        play();
      }
    });
  });

  if (window.__radioPlayerInit) {
    // Already set up on a prior visit - just sync any freshly-rendered button's
    // icon with the real (persisted) <audio> element's playback state.
    var state = audio.paused ? "paused" : "playing";
    fresh.forEach(function (el) { applyState(el, state); });
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
