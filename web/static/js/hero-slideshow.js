// Home's news carousel: crossfades between slides on a timer, with dot and
// prev/next controls. Autoplay pauses while the pointer or keyboard focus is
// inside the section, and never starts at all under prefers-reduced-motion.
(function () {
  const section = document.querySelector("[data-hero-slideshow]");
  if (!section) return;

  const slides = Array.from(section.querySelectorAll(".hero-slide"));
  const dots = Array.from(section.querySelectorAll(".hero-dot"));
  const prev = section.querySelector("[data-hero-prev]");
  const next = section.querySelector("[data-hero-next]");
  if (slides.length === 0) return;

  let current = 0;
  let timer = null;

  function showSlide(i) {
    current = (i + slides.length) % slides.length;
    slides.forEach((slide, idx) => {
      const active = idx === current;
      slide.classList.toggle("pointer-events-none", !active);
      slide.classList.toggle("opacity-0", !active);
      // Hidden slides stay in the DOM for the crossfade, so take them out of the
      // tab order rather than letting focus land on an invisible link. An admin
      // image slide with no link is an href-less <a>: there is nothing to
      // activate, so it never enters the tab order either.
      slide.setAttribute("aria-hidden", active ? "false" : "true");
      slide.tabIndex = active && slide.hasAttribute("href") ? 0 : -1;
    });
    dots.forEach((dot, idx) => {
      const active = idx === current;
      dot.classList.toggle("bg-white", active);
      dot.classList.toggle("bg-white/40", !active);
      if (active) {
        dot.setAttribute("aria-current", "true");
      } else {
        dot.removeAttribute("aria-current");
      }
    });
  }

  function startAutoplay() {
    stopAutoplay();
    if (slides.length <= 1) return;
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
    timer = setInterval(() => showSlide(current + 1), 6000);
  }

  function stopAutoplay() {
    if (timer) {
      clearInterval(timer);
      timer = null;
    }
  }

  function goTo(i) {
    showSlide(i);
    startAutoplay();
  }

  dots.forEach((dot, idx) => dot.addEventListener("click", () => goTo(idx)));
  if (prev) prev.addEventListener("click", () => goTo(current - 1));
  if (next) next.addEventListener("click", () => goTo(current + 1));

  section.addEventListener("mouseenter", stopAutoplay);
  section.addEventListener("focusin", stopAutoplay);
  section.addEventListener("mouseleave", startAutoplay);
  section.addEventListener("focusout", startAutoplay);

  // Turbo swaps the whole <main>, but a cached restore can leave the old timer
  // running against detached nodes.
  document.addEventListener("turbo:before-render", stopAutoplay, { once: true });

  showSlide(0);
  startAutoplay();
})();
