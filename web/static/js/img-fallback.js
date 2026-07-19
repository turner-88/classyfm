(function () {
  document.querySelectorAll("img.js-img-fade").forEach(function (img) {
    if (img.dataset.fallbackBound) return;
    img.dataset.fallbackBound = "1";

    function reveal() {
      img.classList.remove("opacity-0");
    }
    function fail() {
      img.remove();
    }

    if (img.complete) {
      if (img.naturalWidth > 0) reveal();
      else fail();
      return;
    }
    img.addEventListener("load", reveal);
    img.addEventListener("error", fail);
  });
})();
