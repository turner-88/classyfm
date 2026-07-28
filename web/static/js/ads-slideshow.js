// Rotates the banners inside any ad slot the admin set to "slideshow" mode.
// Structurally the same as hero-slideshow.js, but generalized: a page can carry
// more than one ad slot, so each [data-ads-slideshow] box gets its own independent
// timer and dots.
(function () {
  const timers = [];

  document.querySelectorAll("[data-ads-slideshow]").forEach(function (box) {
    const slides = Array.from(box.querySelectorAll(".ads-slide"));
    const dots = Array.from(box.querySelectorAll(".ads-dot"));
    if (slides.length <= 1) return;

    const interval = parseInt(box.dataset.adsInterval, 10) || 6000;
    let current = 0;
    let timer = null;

    function showSlide(i) {
      current = i;
      slides.forEach((slide, idx) => {
        slide.classList.toggle("pointer-events-none", idx !== i);
        slide.classList.toggle("opacity-0", idx !== i);
      });
      dots.forEach((dot, idx) => {
        dot.classList.toggle("bg-brand", idx === i);
        dot.classList.toggle("bg-gray-300", idx !== i);
      });
    }

    function startAutoplay() {
      stopAutoplay();
      if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
      timer = setInterval(() => showSlide((current + 1) % slides.length), interval);
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
