// Drives the mid-article image slideshow on a Hot Release detail page. Structurally
// the same as ads-slideshow.js (multi-instance, crossfade, dots, autoplay that
// respects reduced-motion and pauses on hover/focus), plus prev/next buttons. Only
// present when an article has more than one middle image; a single image renders as
// a static plate with no slideshow markup.
(function () {
  const timers = [];

  document.querySelectorAll("[data-article-slideshow]").forEach(function (box) {
    const slides = Array.from(box.querySelectorAll(".article-slide"));
    const dots = Array.from(box.querySelectorAll(".article-dot"));
    if (slides.length <= 1) return;

    const interval = parseInt(box.dataset.articleInterval, 10) || 6000;
    let current = 0;
    let timer = null;

    function showSlide(i) {
      current = (i + slides.length) % slides.length;
      slides.forEach((slide, idx) => {
        slide.classList.toggle("pointer-events-none", idx !== current);
        slide.classList.toggle("opacity-0", idx !== current);
      });
      dots.forEach((dot, idx) => {
        dot.classList.toggle("bg-brand", idx === current);
        dot.classList.toggle("bg-gray-300", idx !== current);
        if (idx === current) dot.setAttribute("aria-current", "true");
        else dot.removeAttribute("aria-current");
      });
    }

    function startAutoplay() {
      stopAutoplay();
      if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
      timer = setInterval(() => showSlide(current + 1), interval);
      timers.push(timer);
    }

    function stopAutoplay() {
      if (timer) {
        clearInterval(timer);
        timer = null;
      }
    }

    dots.forEach((dot, idx) => {
      dot.addEventListener("click", () => {
        showSlide(idx);
        startAutoplay();
      });
    });

    box.querySelectorAll("[data-article-prev]").forEach((btn) => {
      btn.addEventListener("click", () => {
        showSlide(current - 1);
        startAutoplay();
      });
    });
    box.querySelectorAll("[data-article-next]").forEach((btn) => {
      btn.addEventListener("click", () => {
        showSlide(current + 1);
        startAutoplay();
      });
    });

    box.addEventListener("mouseenter", stopAutoplay);
    box.addEventListener("focusin", stopAutoplay);
    box.addEventListener("mouseleave", startAutoplay);
    box.addEventListener("focusout", startAutoplay);

    startAutoplay();
  });

  // Turbo swaps the whole <body> and re-runs this script on every navigation, so
  // clear this page's timers before the next render or they keep ticking against
  // detached nodes - once per navigation, compounding.
  if (timers.length) {
    document.addEventListener(
      "turbo:before-render",
      function () {
        timers.forEach(clearInterval);
      },
      { once: true }
    );
  }
})();
