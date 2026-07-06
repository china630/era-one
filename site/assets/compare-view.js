/* Head-to-head compare pages — lang-aware, per product family */
(function () {
  var CMP = window.ERA_COMPARE;
  if (!CMP) return;

  var state = { path: null, container: null, family: "control", item: null };

  function t(k) {
    var I = window.ERA_I18N || {};
    var lang = document.documentElement.lang || "en";
    var dict = I[lang] || I.en || {};
    return dict[k] != null ? dict[k] : (I.en && I.en[k] != null ? I.en[k] : k);
  }

  function currentLang() {
    return document.documentElement.lang || localStorage.getItem("era-lang") || "en";
  }

  function langChain(lang) {
    if (lang === "ru") return ["ru", "en"];
    if (lang === "tr" || lang === "ar") return [lang, "en", "ru"];
    return [lang, "en", "ru"];
  }

  function url(lang, path) {
    return new URL(CMP.DS + lang + "/" + path, window.location.href).href;
  }

  function fetchCompare(path) {
    var langs = langChain(currentLang());
    var i = 0;
    function tryNext() {
      if (i >= langs.length) return Promise.reject(new Error("not found"));
      var lang = langs[i++];
      return fetch(url(lang, path)).then(function (r) {
        if (r.ok) return r.text();
        return tryNext();
      });
    }
    return tryNext();
  }

  function extractBody(html) {
    var doc = new DOMParser().parseFromString(html, "text/html");
    var b = doc.querySelector(".body");
    return b ? b.innerHTML : "<p>Content not found.</p>";
  }

  function loadInto(container, path) {
    state.path = path;
    state.container = container;
    container.innerHTML = '<p class="ds-loading">' + t("ds.loading") + '</p>';
    return fetchCompare(path)
      .then(function (html) {
        container.innerHTML = extractBody(html);
        var h1 = container.querySelector("h1");
        if (h1) document.title = h1.textContent.trim() + " | ERA One";
      })
      .catch(function () {
        container.innerHTML = '<p class="ds-error">' + t("ds.error") + '</p>';
      });
  }

  function familyLabel(family) {
    for (var i = 0; i < CMP.FAMS.length; i++) {
      if (CMP.FAMS[i].key === family) return t(CMP.FAMS[i].labelKey) || CMP.FAMS[i].name;
    }
    return CMP.familyName(family);
  }

  function vsTitle(item) {
    return familyLabel(item.family) + " vs " + item.name;
  }

  function renderTabs(activeFamily) {
    return '<div class="compare-tabs">' + CMP.FAMS.map(function (f) {
      var href = "compare.html?family=" + f.key;
      return '<a class="compare-tab' + (f.key === activeFamily ? " active" : "") + '" href="' + href + '">' +
        (t(f.labelKey) || f.name) + "</a>";
    }).join("") + "</div>";
  }

  function renderList(container, family) {
    var list = CMP.byFamily(family).map(function (it) {
      return '<a class="mod-link" href="compare.html?family=' + family + "&vs=" + it.id + '"><b>' +
        vsTitle(it) + '</b><span>→</span></a>';
    }).join("");
    container.innerHTML = list;
  }

  function setLead(family) {
    var lead = document.querySelector(".hero.subhero .lead");
    if (!lead) return;
    var key = "compare.lead." + family;
    var v = t(key);
    if (v !== key) lead.innerHTML = v;
    else lead.innerHTML = t("compare.lead");
    lead.style.display = "";
  }

  document.addEventListener("DOMContentLoaded", function () {
    var root = document.getElementById("compare-root");
    if (!root) return;

    var params = new URLSearchParams(location.search);
    var vs = params.get("vs");
    var family = params.get("family") || "control";
    var item = vs ? CMP.find(vs) : null;
    if (item) family = item.family;
    if (CMP.byFamily(family).length === 0) family = "control";
    state.family = family;

    var content = document.getElementById("compare-content");
    var list = document.getElementById("compare-list");
    var pdfBtn = document.getElementById("compare-pdf-btn");
    var titleEl = document.getElementById("compare-title");
    var tabsHost = document.getElementById("compare-tabs");

    if (tabsHost && !vs) tabsHost.innerHTML = renderTabs(family);

    if (!vs) {
      if (titleEl) titleEl.textContent = t("compare.h1");
      setLead(family);
      if (list) renderList(list, family);
      return;
    }

    var item = CMP.find(vs);
    if (!item) {
      root.innerHTML = '<div class="wrap"><p class="ds-error">' + t("ds.notFound") + '</p><a href="compare.html">' + t("compare.back") + '</a></div>';
      return;
    }
    state.item = item;
    state.family = item.family;
    family = item.family;

    if (titleEl) titleEl.textContent = vsTitle(item);

    var crumbSep = document.getElementById("compare-crumb-sep2");
    var crumbVs = document.getElementById("compare-crumb-vs");
    if (crumbSep) crumbSep.hidden = false;
    if (crumbVs) {
      crumbVs.hidden = false;
      crumbVs.textContent = vsTitle(item);
    }

    if (pdfBtn) {
      pdfBtn.addEventListener("click", function () {
        var lang = langChain(currentLang())[0];
        window.open(url(lang, item.ds), "_blank", "noopener");
      });
    }
    if (content) loadInto(content, item.ds);
  });

  window.addEventListener("era-lang-changed", function () {
    if (state.container && state.path) loadInto(state.container, state.path);
    if (state.item) {
      var titleEl = document.getElementById("compare-title");
      if (titleEl) titleEl.textContent = vsTitle(state.item);
      var crumbVs = document.getElementById("compare-crumb-vs");
      if (crumbVs) crumbVs.textContent = vsTitle(state.item);
    }
  });
})();
