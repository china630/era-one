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

  function enDict() { return (window.ERA_CALC_I18N || {}).en || {}; }

  function lookupNested(obj, dottedKey) {
    var parts = dottedKey.split(".");
    var o = obj;
    for (var i = 0; i < parts.length; i++) {
      if (!o) return null;
      o = o[parts[i]];
    }
    return o != null ? o : null;
  }

  function ssotModule(key) {
    var pr = window.ERA_PRICING;
    if (!pr) return null;
    if (pr.modules && pr.modules[key]) return pr.modules[key];
    if (pr.product_lines) {
      for (var lk in pr.product_lines) {
        var pl = pr.product_lines[lk];
        if (pl.modules && pl.modules[key]) return pl.modules[key];
      }
    }
    return null;
  }

  function moduleTitle(key) {
    var v = t("moduleTitle." + key);
    if (v && v.indexOf("moduleTitle.") !== 0) return v;
    v = lookupNested(enDict(), "moduleTitle." + key);
    if (v != null) return v;
    var m = ssotModule(key);
    return (m && m.title) || key;
  }

  function addonTitle(key) {
    var v = t("addonTitle." + key);
    if (v && v.indexOf("addonTitle.") !== 0) return v;
    v = lookupNested(enDict(), "addonTitle." + key);
    if (v != null) return v;
    return key;
  }

  function bundleTitle(key, fallback) {
    var v = t("bundleTitle." + key);
    if (v && v.indexOf("bundleTitle.") !== 0) return v;
    v = lookupNested(enDict(), "bundleTitle." + key);
    if (v != null) return v;
    return fallback || key;
  }

  function termLabel(key, fallback) {
    var v = t("termLabel." + key);
    if (v && v.indexOf("termLabel.") !== 0) return v;
    v = lookupNested(enDict(), "termLabel." + key);
    if (v != null) return v;
    return fallback || key;
  }

  function regionLabel(regionKey) {
    return regionKey === "cis" ? t("regionCis") : t("regionEu");
  }

  function moduleDesc(key) {
    var v = t("moduleDesc." + key);
    if (v && v.indexOf("moduleDesc.") !== 0) return v;
    v = lookupNested(enDict(), "moduleDesc." + key);
    if (v != null) return v;
    var m = ssotModule(key);
    return (m && m.desc) || "";
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
  window.ERA_LANG = {
    t: t,
    catLabel: catLabel,
    moduleDesc: moduleDesc,
    moduleTitle: moduleTitle,
    addonTitle: addonTitle,
    bundleTitle: bundleTitle,
    termLabel: termLabel,
    regionLabel: regionLabel,
    getLang: getLang,
    setLang: setLang
  };

  window.addEventListener("era-lang-changed", function (ev) {
    if (ev.detail && ev.detail.lang) setLang(ev.detail.lang);
  });

  applyStatic();
})();
