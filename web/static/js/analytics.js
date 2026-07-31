// Google Analytics 4 loader. The site's CSP has no 'unsafe-inline', so Google's stock
// snippet can't be pasted into the layout — the measurement ID arrives on a data
// attribute instead and this file does the bootstrap. The layout omits the tag entirely
// when GA_MEASUREMENT_ID is unset, so reaching here without an id shouldn't happen; bail
// rather than request a tag for the empty property.
//
// This must stay in <head>: Turbo re-executes <script> elements in a swapped <body> on
// every navigation, which would re-register the turbo:load listener below and double
// count. Head scripts are merged by src and run once per session.
(function () {
  var el = document.currentScript;
  var id = el && el.getAttribute("data-ga-id");
  if (!id) return;

  var loader = document.createElement("script");
  loader.async = true;
  loader.src = "https://www.googletagmanager.com/gtag/js?id=" + encodeURIComponent(id);
  document.head.appendChild(loader);

  window.dataLayer = window.dataLayer || [];
  function gtag() {
    window.dataLayer.push(arguments);
  }
  window.gtag = gtag;
  gtag("js", new Date());
  // Turbo Drive swaps pages without a reload, so gtag's own automatic page_view would
  // only ever fire on the first load. Turn it off and send one per turbo:load, which
  // fires on the initial page too — so every view is counted exactly once.
  gtag("config", id, { send_page_view: false });

  document.addEventListener("turbo:load", function () {
    gtag("event", "page_view", {
      page_title: document.title,
      page_location: window.location.href,
      page_path: window.location.pathname + window.location.search,
    });
  });
})();
