// Running text for lines too wide for their container - the floating player's song
// and artist lines, and the TikTok card's tagline, all of which sit in a fixed-width
// card (see .widget-card) where a long song title would otherwise just end in an
// ellipsis. Markup opts in with `.marquee` on the line plus a `.marquee-inner` span
// around its text; everything else (whether it scrolls at all, how far, how long)
// is measured here and handed to CSS as custom properties.
//
// Nothing scrolls until this file confirms the line actually overflows, and a
// visitor who asked for reduced motion keeps the ellipsis instead - which is the
// point of the is-marquee class rather than an unconditional animation, since the
// global prefers-reduced-motion rule would otherwise crush the animation to 0.01ms
// and freeze the line at its fully-scrolled frame.
//
// Loaded before radio.js and tiktok-live.js: they own the text those lines show and
// call setText() instead of writing textContent, so every poll re-measures.
(function () {
  "use strict";

  // Pixels per second, and the share of the cycle each travel leg gets (the
  // keyframes in tailwind.config.js spend 36% scrolling each way and hold for the
  // rest). Long titles are therefore slower in wall-clock terms but read at the
  // same speed. The clamp keeps a two-word overflow from flickering past and a
  // pathological title from taking a minute.
  var SPEED = 45;
  var TRAVEL_SHARE = 0.36;
  var MIN_SECONDS = 6;
  var MAX_SECONDS = 30;
  // Sub-pixel slack: scrollWidth/clientWidth are rounded, and a line overflowing
  // by a hair would scroll imperceptibly for no reason.
  var SLACK = 2;

  var reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)");

  function stop(el) {
    el.classList.remove("is-marquee");
    el.style.removeProperty("--marquee-shift");
    el.style.removeProperty("--marquee-duration");
  }

  // sync measures one .marquee element and turns its animation on or off.
  //
  // The inner span stays inline-block with overflow:hidden in both states (see
  // tailwind.css), so its scrollWidth is the true content width whether or not an
  // animation is currently running - no need to strip is-marquee before measuring,
  // which is what keeps the ResizeObserver below from re-triggering on its own
  // writes.
  function sync(el) {
    var inner = el.querySelector(".marquee-inner");
    if (!inner) return;

    // The card is collapsed to its bubble (display:none), so there is nothing to
    // measure. Leave whatever state it had - the observer fires again on restore.
    if (!el.clientWidth) return;

    var shift = inner.scrollWidth - el.clientWidth;
    if (shift <= SLACK || reduceMotion.matches) {
      stop(el);
      return;
    }

    var seconds = shift / SPEED / TRAVEL_SHARE;
    if (seconds < MIN_SECONDS) seconds = MIN_SECONDS;
    if (seconds > MAX_SECONDS) seconds = MAX_SECONDS;

    // CSSOM writes, not a style="" attribute: the site's CSP has no 'unsafe-inline'
    // for style-src, and that restriction covers attributes parsed out of HTML, not
    // this.
    el.style.setProperty("--marquee-shift", "-" + shift + "px");
    el.style.setProperty("--marquee-duration", seconds.toFixed(2) + "s");
    el.classList.add("is-marquee");
  }

  function syncAll() {
    document.querySelectorAll(".marquee").forEach(sync);
  }

  // setText is the call site's entry point: writes the line's text and re-measures
  // it in one step. Falls back to the element itself when the markup has no inner
  // span, so a caller can point it at a plain line without breaking.
  function setText(el, text) {
    if (!el) return;
    var inner = el.querySelector(".marquee-inner");
    (inner || el).textContent = text;
    if (inner) sync(el);
  }

  window.ClassyMarquee = { sync: sync, syncAll: syncAll, setText: setText };

  // Observe every marquee line for width changes. This is what covers a viewport
  // resize, the sm: breakpoint widening the card, and a collapsed card being
  // restored (display:none reports 0 and the restore reports the real width). Text
  // changes don't move the container, so those go through setText instead.
  //
  // Turbo re-executes this file on every in-site visit, so the observer is created
  // once and only the element scan repeats - observe() on an already-observed
  // element is a no-op, which makes the rescan safe.
  if (!window.__classyMarqueeObserver && window.ResizeObserver) {
    window.__classyMarqueeObserver = new ResizeObserver(function (entries) {
      entries.forEach(function (entry) { sync(entry.target); });
    });
  }
  var observer = window.__classyMarqueeObserver;
  if (observer) {
    document.querySelectorAll(".marquee").forEach(function (el) {
      observer.observe(el);
    });
  }

  if (window.__classyMarqueeInit) {
    // Later visit: the observer above already picked up anything newly rendered,
    // and a fallback measure covers browsers without ResizeObserver.
    if (!observer) syncAll();
    return;
  }
  window.__classyMarqueeInit = true;

  // A line measured in the fallback font can be wider or narrower than the same
  // line in Outfit, so measure again once the webfonts are in.
  if (document.fonts && document.fonts.ready) {
    document.fonts.ready.then(syncAll).catch(function () {});
  }

  // Toggling the OS setting mid-session should stop or start the lines to match.
  if (reduceMotion.addEventListener) {
    reduceMotion.addEventListener("change", syncAll);
  }

  if (!observer) window.addEventListener("resize", syncAll);

  syncAll();
})();
