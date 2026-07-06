/* ERA One portal — language switcher (RU/EN). */
(function () {
  "use strict";
  var STORAGE_KEY = "era-portal-lang";
  var lang = localStorage.getItem(STORAGE_KEY) || "ru";
  if (lang !== "ru" && lang !== "en") lang = "ru";

  function t(key) {
    var parts = key.split(".");
    var o = window.ERA_I18N[lang];
    for (var i = 0; i < parts.length; i++) {
      if (!o) return key;
      o = o[parts[i]];
    }
    return o != null ? o : key;
  }

  function catLabel(key) {
    var map = { secops: "catSecops", itops: "catItops", identity: "catIdentity", platform: "catPlatform" };
    return t(map[key] || key);
  }

  function moduleDesc(key) {
    return t("moduleDesc." + key) || (window.ERA_PRICING && window.ERA_PRICING.modules[key] && window.ERA_PRICING.modules[key].desc) || "";
  }

  function applyStatic() {
    document.documentElement.lang = lang;
    document.title = t("metaTitle");
    var meta = document.querySelector('meta[name="description"]');
    if (meta) meta.setAttribute("content", t("metaDesc"));

    document.querySelectorAll("[data-i18n]").forEach(function (el) {
      var k = el.getAttribute("data-i18n");
      var val = t(k);
      if (el.tagName === "INPUT" || el.tagName === "TEXTAREA") el.placeholder = val;
      else el.innerHTML = val;
    });

    document.querySelectorAll(".lang-switch button").forEach(function (b) {
      b.classList.toggle("active", b.dataset.lang === lang);
      b.setAttribute("aria-pressed", b.dataset.lang === lang ? "true" : "false");
    });
  }

  function setLang(l) {
    if (l !== "ru" && l !== "en") return;
    lang = l;
    localStorage.setItem(STORAGE_KEY, lang);
    applyStatic();
    document.dispatchEvent(new CustomEvent("eralangchange", { detail: { lang: lang } }));
  }

  function getLang() { return lang; }

  document.addEventListener("DOMContentLoaded", function () {
    document.querySelectorAll(".lang-switch button").forEach(function (b) {
      b.onclick = function () { setLang(b.dataset.lang); };
    });
    applyStatic();
  });

  window.ERA_LANG = { t: t, catLabel: catLabel, moduleDesc: moduleDesc, getLang: getLang, setLang: setLang };
})();
