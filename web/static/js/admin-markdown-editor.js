// Mounts a Toast UI Editor (WYSIWYG) over any [data-markdown-editor] textarea and
// keeps the textarea in sync so the plain form POST still submits Markdown in the
// `body` field. Storage and the whole server side stay Markdown-only; this is a
// progressive enhancement — with the vendored bundle absent or failed, the raw
// textarea below stays visible and editable.
//
// Loaded only on the legal edit page (see admin/legal_form.html's admin-scripts
// block). The vendored bundle (toastui-editor-all.min.js) runs first, so the
// global `toastui` exists by the time this deferred script executes.
(function () {
  "use strict";

  var textarea = document.querySelector("textarea[data-markdown-editor]");
  if (!textarea || !window.toastui || !window.toastui.Editor) return;

  var form = textarea.form;

  // Host element for the editor, placed where the textarea sits; the textarea is
  // hidden (not removed) so its value still posts as `body`.
  var host = document.createElement("div");
  textarea.parentNode.insertBefore(host, textarea);
  // Inline display:none, not the `hidden` attribute: .form-textarea applies
  // `display:block` (a class, specificity 0,1,0) which overrides Tailwind
  // preflight's `[hidden]:where(...){display:none}` (zero specificity via :where),
  // so `hidden` alone leaves the raw textarea visible under the editor. The
  // textarea stays in the form, so its value still posts as `body`.
  textarea.style.display = "none";

  var editor = new toastui.Editor({
    el: host,
    height: "32rem",
    initialEditType: "wysiwyg",
    previewStyle: "tab",
    hideModeSwitch: false,
    initialValue: textarea.value,
    // Toast UI otherwise beacons Google Analytics on init, which both trips the
    // site CSP (connect-src) and is unwanted.
    usageStatistics: false,
    // Restrict to the formatting our sanitizer (bluemonday UGCPolicy) keeps and
    // .prose-article styles: headings, bold, italic, lists, links, blockquote.
    toolbarItems: [
      ["heading", "bold", "italic"],
      ["ul", "ol", "quote"],
      ["link"],
    ],
  });

  // Mirror the editor's Markdown back into the textarea on every change, and fire
  // an input event so admin-forms.js's unsaved-changes guard notices the edit.
  function sync() {
    textarea.value = editor.getMarkdown();
    textarea.dispatchEvent(new Event("input", { bubbles: true }));
  }
  editor.on("change", sync);

  // Final sync on submit, in the capture phase so it runs before admin-forms.js's
  // bubbling submit handler clears the dirty flag.
  if (form) {
    form.addEventListener(
      "submit",
      function () {
        textarea.value = editor.getMarkdown();
      },
      true
    );
  }
})();
