/* Pricing calculator modal — Control, Communications, Office (lazy mount). */
(function () {
  "use strict";
  var mounted = {};

  function lineKey() {
    return document.body.getAttribute("data-family") || "control";
  }

  function mountCalc(body) {
    var key = lineKey();
    if (mounted[key]) return;
    if (key === "control" && window.ERA_mountControlCalc) {
      window.ERA_mountControlCalc(body);
    } else if ((key === "communications" || key === "office") && window.ERA_mountProductCalc) {
      window.ERA_mountProductCalc(body, key);
    }
    mounted[key] = true;
  }

  function openModal(overlay) {
    overlay.hidden = false;
    overlay.setAttribute("aria-hidden", "false");
    document.body.classList.add("modal-open");
    var body = document.getElementById("calc-modal-body");
    if (body) mountCalc(body);
    var closeBtn = document.getElementById("calc-modal-close");
    if (closeBtn) closeBtn.focus();
  }

  function closeModal(overlay) {
    overlay.hidden = true;
    overlay.setAttribute("aria-hidden", "true");
    document.body.classList.remove("modal-open");
    var openBtn = document.getElementById("ds-calc-btn");
    if (openBtn) openBtn.focus();
  }

  document.addEventListener("DOMContentLoaded", function () {
    var openBtn = document.getElementById("ds-calc-btn");
    var overlay = document.getElementById("calc-modal");
    if (!openBtn || !overlay) return;
    openBtn.addEventListener("click", function () { openModal(overlay); });
    var closeBtn = document.getElementById("calc-modal-close");
    if (closeBtn) closeBtn.addEventListener("click", function () { closeModal(overlay); });
  });
})();
