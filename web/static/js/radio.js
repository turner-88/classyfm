// Radio playback + live now-playing metadata. Renders into every element carrying a
// js-np-* class (floating player pill, /live page) and binds every .js-radio-toggle
// button (play/pause). Turbo re-executes this body <script> tag on every in-site
// navigation, so setup is split into a one-time part (audio element listeners + the
// poll loop, guarded by window.__radioPlayerInit since #radio-audio lives inside the
// floating player's data-turbo-permanent wrapper and is never replaced) and a
// per-visit part that always runs: binding any freshly-rendered .js-radio-toggle,
// such as /live's own button, which Turbo destroys and recreates on every visit,
// and re-asserting the current playback state over the new <body>.
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

  // currentState reports playback as the persistent <audio> element actually sees
  // it. Not paused but not yet buffered is the spinner, not the pause icon - a
  // visit landing in the join-the-stream gap would otherwise claim playback had
  // already started.
  function currentState() {
    if (audio.paused) return "paused";
    return audio.readyState < 3 ? "loading" : "playing"; // < HAVE_FUTURE_DATA
  }

  function setState(state) {
    document.querySelectorAll(".js-radio-toggle, .js-radio-state").forEach(function (el) {
      applyState(el, state);
    });
    // One class drives every playing-only affordance (equalizer bars, the play
    // button's ripple arcs) via CSS, so widgets don't each need a hook.
    //
    // It goes on <html>, not <body>: Turbo replaces <body> wholesale on every
    // visit with the server's copy, which never carries this class, so the
    // animations used to die on the first in-site navigation while the stream
    // kept playing. Turbo never touches documentElement (only its lang/dir
    // attributes), so the class - and with it the running CSS animations on the
    // data-turbo-permanent floating player - survives a visit uninterrupted,
    // with no restart to re-trigger. It also keeps the class out of Turbo's
    // snapshot cache, which clones <body>: a page cached mid-playback used to
    // come back animating on a back-navigation even once audio was paused.
    document.documentElement.classList.toggle("is-playing", state === "playing");
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

  // setLine writes one of the now-playing text lines. It routes through
  // marquee.js where that's loaded, since the floating player's song/artist lines
  // wrap their text in a .marquee-inner span and have to be re-measured after every
  // write - a plain textContent here would delete the span. /live's copies aren't
  // marquees and pass straight through either way.
  function setLine(el, text) {
    if (window.ClassyMarquee) {
      window.ClassyMarquee.setText(el, text);
      return;
    }
    el.textContent = text;
  }

  // announcerSig tracks which announcers are currently rendered as chips so the poll
  // rebuilds them only when the set changes (it changes at program boundaries, not
  // every 15s) - re-creating identical chips would re-fetch avatars and flicker. Seeded
  // from the server-rendered chips' /broadcasters/<slug> hrefs so an unchanged first
  // poll leaves them (and their img-fallback fade) untouched.
  var announcerSig = (function () {
    var c = document.querySelector(".js-np-announcers");
    if (!c) return null;
    return Array.prototype.map.call(c.querySelectorAll("a"), function (link) {
      return (link.getAttribute("href") || "").split("/broadcasters/")[1] || "";
    }).join("|");
  })();

  // buildAnnouncerChip creates one on-air announcer chip (avatar + name), matching the
  // server-rendered markup in live.html. a is {name, photo_url, slug}. The avatar image
  // is painted directly (no js-img-fade/opacity-0): img-fallback.js only binds elements
  // present at load, so a chip built here would stay invisible under opacity-0.
  function buildAnnouncerChip(a) {
    var link = document.createElement("a");
    link.href = "/broadcasters/" + a.slug;
    link.className = "inline-flex items-center gap-2 rounded-full bg-white/10 py-1 pl-1 pr-3 text-white transition-colors hover:bg-white/20";

    var avatar = document.createElement("span");
    avatar.className = "relative flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-full bg-gradient-to-br from-brand to-brand-800 text-white/40";
    avatar.innerHTML = '<svg class="h-5 w-5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>';
    if (a.photo_url) {
      var img = document.createElement("img");
      img.src = a.photo_url;
      img.alt = "";
      img.loading = "lazy";
      img.decoding = "async";
      img.className = "absolute inset-0 h-full w-full object-cover";
      avatar.appendChild(img);
    }
    link.appendChild(avatar);

    var name = document.createElement("span");
    name.className = "text-sm font-semibold";
    name.textContent = a.name;
    link.appendChild(name);
    return link;
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
      setLine(el, firstText(titles, el.dataset.stationName));
    });
    document.querySelectorAll(".js-np-artist").forEach(function (el) {
      setLine(el, firstText(subtitles, el.dataset.stationSlogan));
    });

    // /live's on-air announcers: avatar + name chips (1-6), rebuilt from np.announcers
    // only when the set changes (see announcerSig). The wrapper hides when nothing is on
    // air or the program has no announcers.
    var announcers = offline ? [] : (np.announcers || []);
    var sig = announcers.map(function (a) { return a.slug; }).join("|");
    if (sig !== announcerSig) {
      announcerSig = sig;
      document.querySelectorAll(".js-np-announcers").forEach(function (container) {
        container.textContent = "";
        announcers.forEach(function (a) { container.appendChild(buildAnnouncerChip(a)); });
      });
    }
    document.querySelectorAll(".js-np-announcers-wrap").forEach(function (el) {
      el.classList.toggle("hidden", announcers.length === 0);
    });

    document.querySelectorAll(".js-np-cover").forEach(function (el) {
      el.src = (!offline && np.cover_url) ? np.cover_url : "/static/img/default-cover.jpg";
    });

    // iTunes Store badge on the cover(s): only when a real iTunes result is showing
    // (cover + store link). Drives the floating player and /live plate from this
    // one poll, same as the .js-np-cover hook above.
    document.querySelectorAll(".js-np-store").forEach(function (el) {
      var show = !offline && np.cover_url && np.store_url;
      el.classList.toggle("hidden", !show);
      if (show) el.href = np.store_url;
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

  document.querySelectorAll(".js-radio-toggle").forEach(function (toggle) {
    if (toggle.dataset.bound) return;
    toggle.dataset.bound = "1";
    toggle.addEventListener("click", toggleClick);
  });

  // .js-radio-state elements need no pass of their own: they carry no click
  // handler (they only reveal a collapsed card, which has to keep working while
  // the stream is off air, so renderNowPlaying never disables them either), and
  // setState below re-queries them by class on every call.

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
    // Already set up on a prior visit. Turbo just swapped in the server's freshly
    // rendered <body>, so re-assert playback state over whatever is now in the
    // document - /live's own play button, the header - from the real (persisted)
    // <audio> element. setState rather than a per-element applyState: it also
    // re-sets the <html> class, so a visit that somehow lands with it out of sync
    // heals itself instead of silently losing every playing-only animation.
    setState(currentState());
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
