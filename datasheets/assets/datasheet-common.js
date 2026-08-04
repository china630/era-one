/* Shared datasheet behaviour: floating "Download PDF" button + logo fallback. */
(function () {
  function ready(fn) {
    if (document.readyState !== "loading") fn();
    else document.addEventListener("DOMContentLoaded", fn);
  }
  ready(function () {
    // Logo fallback (banner PNG may be absent in the repo)
    document.querySelectorAll('img[src*="era-one-logo-banner"]').forEach(function (img) {
      img.addEventListener("error", function () {
        img.onerror = null;
        img.src = "../assets/era-one-logo-banner.svg";
      });
      // trigger a load check for cached-broken images
      if (img.complete && img.naturalWidth === 0) { img.onerror = null; img.src = "../assets/era-one-logo-banner.svg"; }
    });

    // Floating Download PDF (uses the browser's print-to-PDF)
    if (!document.querySelector(".pdf-fab")) {
      var btn = document.createElement("button");
      btn.className = "pdf-fab";
      btn.type = "button";
      btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M12 3v12m0 0l-4-4m4 4l4-4M5 21h14"/></svg><span>Download PDF</span>';
      btn.addEventListener("click", function () { window.print(); });
      document.body.appendChild(btn);
    }
  });
})();
