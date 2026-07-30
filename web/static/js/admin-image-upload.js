// Browser-side image compression for the admin's five upload fields (program
// image, broadcaster photo, hot-release image, about banner, ad banner).
//
// Admins upload straight off a phone: 8-15 MB, 4000px+. Nothing on the site
// renders wider than ~1600px, so shipping the original wastes the admin's
// upload time, the server's disk, and every visitor's bandwidth - and it used
// to fail outright against the request size ceiling. Here the file is decoded,
// scaled to fit the field's [data-max-edge], re-encoded as WebP, and swapped
// back into the input's FileList before the form posts. The server sees an
// ordinary small upload and needs no special handling.
//
// Hook: <input type="file" data-image-upload data-max-edge="1600">, optionally
// paired with [data-image-upload-status] and [data-image-upload-preview]
// siblings inside the same form section.
//
// Two constraints worth knowing before editing this file:
//
//   - The CSP in internal/middleware/security.go allows img-src 'self' https:
//     data: - note the absence of blob:. URL.createObjectURL() previews are
//     blocked outright, so previews go through FileReader/data: URLs and
//     decoding goes through createImageBitmap(), which takes the Blob directly
//     and never needs a URL at all.
//   - Compression is async, so a fast Save click would post the original file.
//     The capture-phase submit listener below holds the submission and replays
//     it once the queue drains.
(function () {
  "use strict";

  var DEFAULT_MAX_EDGE = 1600;
  var QUALITY = 0.85;
  // Below this, re-encoding buys little and can even lose: leave the file alone
  // if it also already fits the target dimensions.
  var SKIP_UNDER_BYTES = 500 * 1024;

  var pending = 0;
  var queuedForm = null;
  var queuedSubmitter = null;

  document.addEventListener("change", function (e) {
    var input = e.target;
    if (!input || !input.matches || !input.matches("input[type=file][data-image-upload]")) return;
    handle(input);
  });

  // Runs in the capture phase so it beats the browser's own submission. A
  // pending compression means input.files still holds the original.
  document.addEventListener(
    "submit",
    function (e) {
      if (pending <= 0) return;
      e.preventDefault();
      queuedForm = e.target;
      // The delete button shares this form via formaction/formmethod, so the
      // submitter has to be carried across or the replay posts to the wrong URL.
      queuedSubmitter = e.submitter || null;
    },
    true
  );

  function handle(input) {
    var file = input.files && input.files[0];
    if (!file) {
      setStatus(input, "");
      return;
    }

    // Animated GIFs cannot survive a canvas round-trip - drawImage grabs one
    // frame - so they are passed through at their original size.
    if (file.type === "image/gif") {
      setStatus(input, fmtBytes(file.size) + " GIF · uploaded as-is so animation is preserved");
      showPreview(input, file);
      return;
    }

    var form = input.form;
    startWork(form);
    setStatus(input, "Compressing…");

    compress(input, file).then(
      function (result) {
        if (result.file !== file && !swapIn(input, result.file)) {
          // No DataTransfer support: the original is still in the input and will
          // upload fine, it is just bigger than it needed to be.
          result.file = file;
          result.compressed = false;
          result.width = result.srcWidth;
          result.height = result.srcHeight;
        }
        setStatus(input, describe(file, result));
        showPreview(input, result.file);
        finishWork(form);
      },
      function () {
        // Decode or encode failed (corrupt file, exotic format, out of memory).
        // The original is untouched and still posts; the server does the real
        // validation anyway.
        setStatus(input, fmtBytes(file.size) + " · uploaded without compression");
        finishWork(form);
      }
    );
  }

  // --- compression ----------------------------------------------------------

  function compress(input, file) {
    var maxEdge = parseInt(input.getAttribute("data-max-edge"), 10) || DEFAULT_MAX_EDGE;

    return decode(file).then(function (src) {
      var sw = src.width || src.naturalWidth;
      var sh = src.height || src.naturalHeight;
      if (!sw || !sh) throw new Error("zero-size image");

      var scale = Math.min(1, maxEdge / Math.max(sw, sh));
      var tw = Math.max(1, Math.round(sw * scale));
      var th = Math.max(1, Math.round(sh * scale));

      if (scale === 1 && file.size < SKIP_UNDER_BYTES) {
        return { file: file, width: sw, height: sh, srcWidth: sw, srcHeight: sh, compressed: false };
      }

      var canvas = document.createElement("canvas");
      canvas.width = tw;
      canvas.height = th;
      var ctx = canvas.getContext("2d");
      ctx.drawImage(src, 0, 0, tw, th);
      if (src.close) src.close();

      return encode(canvas, "image/webp", QUALITY)
        .then(function (blob) {
          if (blob && blob.type === "image/webp") return { blob: blob, ext: ".webp" };
          // Browser has no WebP encoder (Safari < 14): toBlob quietly hands back
          // PNG, which is typically larger than the source. Re-encode as JPEG,
          // painting white *behind* the drawing first - JPEG has no alpha, and
          // transparent pixels would otherwise come out black.
          ctx.globalCompositeOperation = "destination-over";
          ctx.fillStyle = "#ffffff";
          ctx.fillRect(0, 0, tw, th);
          ctx.globalCompositeOperation = "source-over";
          return encode(canvas, "image/jpeg", QUALITY).then(function (jpeg) {
            return { blob: jpeg, ext: ".jpg" };
          });
        })
        .then(function (out) {
          // Re-encoding a already-small file can come out bigger; only accept the
          // result when it actually helped or when we genuinely downscaled.
          if (!out.blob || (scale === 1 && out.blob.size >= file.size)) {
            return { file: file, width: sw, height: sh, srcWidth: sw, srcHeight: sh, compressed: false };
          }
          var name = baseName(file.name) + out.ext;
          return {
            file: new File([out.blob], name, { type: out.blob.type, lastModified: Date.now() }),
            width: tw,
            height: th,
            srcWidth: sw,
            srcHeight: sh,
            compressed: true
          };
        });
    });
  }

  // decode prefers createImageBitmap: it accepts the File directly (no blob:
  // URL, which the CSP would block) and "from-image" applies the EXIF rotation
  // tag, without which portrait phone photos land on their side.
  function decode(file) {
    return new Promise(function (resolve, reject) {
      if (typeof window.createImageBitmap === "function") {
        var p;
        try {
          p = window.createImageBitmap(file, { imageOrientation: "from-image" });
        } catch (err) {
          p = null;
        }
        if (p && typeof p.then === "function") {
          p.then(resolve, function () {
            decodeViaDataUrl(file).then(resolve, reject);
          });
          return;
        }
      }
      decodeViaDataUrl(file).then(resolve, reject);
    });
  }

  function decodeViaDataUrl(file) {
    return readDataUrl(file).then(function (url) {
      return new Promise(function (resolve, reject) {
        var img = new Image();
        img.onload = function () {
          resolve(img);
        };
        img.onerror = function () {
          reject(new Error("decode failed"));
        };
        img.src = url;
      });
    });
  }

  function encode(canvas, type, quality) {
    return new Promise(function (resolve) {
      if (typeof canvas.toBlob !== "function") {
        resolve(null);
        return;
      }
      canvas.toBlob(function (blob) {
        resolve(blob);
      }, type, quality);
    });
  }

  function swapIn(input, file) {
    if (typeof window.DataTransfer !== "function") return false;
    try {
      var dt = new DataTransfer();
      dt.items.add(file);
      input.files = dt.files;
      return true;
    } catch (err) {
      return false;
    }
  }

  // --- submit gating --------------------------------------------------------

  function startWork(form) {
    pending++;
    setBusy(form, true);
  }

  function finishWork(form) {
    pending = Math.max(0, pending - 1);
    if (pending > 0) return;
    setBusy(form, false);
    if (!queuedForm) return;
    var f = queuedForm;
    var submitter = queuedSubmitter;
    queuedForm = null;
    queuedSubmitter = null;
    // requestSubmit re-fires the submit event, but pending is 0 now so the
    // listener above lets it through.
    if (typeof f.requestSubmit === "function") f.requestSubmit(submitter || undefined);
    else f.submit();
  }

  // Buttons wrap an icon <span> alongside their label, so the label is left
  // alone; disabling is the whole signal, and the per-field status line already
  // says "Compressing...".
  function setBusy(form, busy) {
    if (!form) return;
    var buttons = form.querySelectorAll("button[type=submit], input[type=submit]");
    Array.prototype.forEach.call(buttons, function (btn) {
      if (busy) {
        if (btn.disabled) return; // already disabled for its own reasons
        btn.disabled = true;
        btn.setAttribute("data-image-upload-busy", "");
        btn.setAttribute("aria-busy", "true");
      } else if (btn.hasAttribute("data-image-upload-busy")) {
        btn.disabled = false;
        btn.removeAttribute("data-image-upload-busy");
        btn.removeAttribute("aria-busy");
      }
    });
  }

  // --- status + preview -----------------------------------------------------

  function describe(original, result) {
    var dims = " · " + result.width + "×" + result.height;
    if (!result.compressed) return fmtBytes(original.size) + dims + " · uploaded as-is";
    return fmtBytes(original.size) + " → " + fmtBytes(result.file.size) + dims;
  }

  function setStatus(input, text) {
    var el = scope(input).querySelector("[data-image-upload-status]");
    if (!el) return;
    el.textContent = text;
    el.classList.toggle("hidden", !text);
  }

  function showPreview(input, file) {
    var img = scope(input).querySelector("[data-image-upload-preview]");
    if (!img) return;
    readDataUrl(file).then(
      function (url) {
        img.src = url;
        img.classList.remove("hidden");
      },
      function () {}
    );
  }

  // The status and preview hooks live in the same form section as the input;
  // fall back to the form so a differently-shaped page still works.
  function scope(input) {
    return input.closest(".form-section") || input.closest(".form-field") || input.form || document;
  }

  function readDataUrl(file) {
    return new Promise(function (resolve, reject) {
      var reader = new FileReader();
      reader.onload = function () {
        resolve(reader.result);
      };
      reader.onerror = function () {
        reject(new Error("read failed"));
      };
      reader.readAsDataURL(file);
    });
  }

  function baseName(name) {
    var dot = (name || "image").lastIndexOf(".");
    var base = dot > 0 ? name.slice(0, dot) : name || "image";
    return base.replace(/[^A-Za-z0-9._-]+/g, "-").slice(0, 60) || "image";
  }

  function fmtBytes(n) {
    if (n < 1024) return n + " B";
    if (n < 1024 * 1024) return Math.round(n / 1024) + " KB";
    return (n / (1024 * 1024)).toFixed(1) + " MB";
  }
})();
