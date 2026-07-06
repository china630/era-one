/* ERA One portal — product cards from SSOT + i18n. */
(function () {
  "use strict";

  function renderProducts() {
    var host = document.getElementById("product-groups");
    if (!host || !window.ERA_PRICING || !window.ERA_MODULE_GROUPS) return;
    var L = window.ERA_LANG;
    var D = window.ERA_PRICING;
    var html = "";

    window.ERA_MODULE_GROUPS.forEach(function (grp) {
      html += '<div class="product-group"><h3 class="cat">' + L.catLabel(grp.key) + '</h3><div class="cards">';
      grp.modules.forEach(function (key) {
        var m = D.modules[key];
        if (!m) return;
        var tag = m.availability === "project"
          ? ' <span class="avail">' + L.t("projectTag") + '</span>' : "";
        html += '<div class="card"><div class="nm">' + m.title + tag + '</div>' +
                '<div class="ds">' + L.moduleDesc(key) + '</div></div>';
      });
      html += "</div></div>";
    });
    host.innerHTML = html;
  }

  document.addEventListener("DOMContentLoaded", renderProducts);
  document.addEventListener("eralangchange", renderProducts);
})();
