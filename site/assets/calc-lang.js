/* Bridge site language (era-lang) to calculator (ERA_LANG / eralangchange). */
(function () {
  "use strict";
  var lang = "en";
  try { lang = localStorage.getItem("era-lang") || "en"; } catch (e) {}

  function dict() {
    var D = window.ERA_CALC_I18N || {};
    return D[lang] || D.en || {};
  }

  function t(key) {
    var parts = key.split(".");
    var o = dict();
    for (var i = 0; i < parts.length; i++) {
      if (!o) return key;
      o = o[parts[i]];
    }
    if (o != null) return o;
    var en = (window.ERA_CALC_I18N || {}).en || {};
    o = en;
    for (var j = 0; j < parts.length; j++) {
      if (!o) return key;
      o = o[parts[j]];
    }
    return o != null ? o : key;
  }

  function catLabel(key) {
    var map = { secops: "catSecops", itops: "catItops", identity: "catIdentity", platform: "catPlatform" };
    return t(map[key] || key);
  }

  function moduleDesc(key) {
    return t("moduleDesc." + key) ||
      (window.ERA_PRICING && window.ERA_PRICING.modules[key] && window.ERA_PRICING.modules[key].desc) ||
      "";
  }

  function applyStatic() {
    var root = document.getElementById("era-calc");
    if (!root) return;
    root.querySelectorAll("[data-i18n-calc]").forEach(function (el) {
      var k = el.getAttribute("data-i18n-calc");
      var val = t(k);
      if (val != null) el.innerHTML = val;
    });
  }

  function setLang(l) {
    if (!l) return;
    lang = l;
    applyStatic();
    document.dispatchEvent(new CustomEvent("eralangchange", { detail: { lang: lang } }));
  }

  function getLang() { return lang; }

  window.ERA_CALC_applyStatic = applyStatic;
  window.ERA_LANG = { t: t, catLabel: catLabel, moduleDesc: moduleDesc, getLang: getLang, setLang: setLang };

  window.addEventListener("era-lang-changed", function (ev) {
    if (ev.detail && ev.detail.lang) setLang(ev.detail.lang);
  });

  applyStatic();
})();
