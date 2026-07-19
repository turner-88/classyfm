(function () {
  const section = document.querySelector("[data-hero-slideshow]");
  if (!section) return;

  const slides = Array.from(section.querySelectorAll(".hero-slide"));
  const dots = Array.from(section.querySelectorAll(".hero-dot"));
  if (slides.length === 0) return;

  let current = 0;
  let timer = null;

  function showSlide(i) {
    current = i;
    slides.forEach((slide, idx) => {
      slide.classList.toggle("pointer-events-none", idx !== i);
      slide.classList.toggle("opacity-0", idx !== i);
    });
    dots.forEach((dot, idx) => {
      dot.classList.toggle("bg-white", idx === i);
      dot.classList.toggle("bg-white/50", idx !== i);
    });
  }

  function startAutoplay() {
    stopAutoplay();
    if (slides.length <= 1) return;
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
    timer = setInterval(() => showSlide((current + 1) % slides.length), 6000);
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

  section.addEventListener("mouseenter", stopAutoplay);
  section.addEventListener("focusin", stopAutoplay);
  section.addEventListener("mouseleave", startAutoplay);
  section.addEventListener("focusout", startAutoplay);

  startAutoplay();
})();
