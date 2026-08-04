/* Load datasheet HTML into site shell; lang-aware; PDF opens print page.
   Respects build-time prerender (data-prerendered) so crawlers see content without JS. */
(function () {
  var CAT = window.ERA_CATALOG;
  if (!CAT) return;

  var state = { path: null, familyKey: null, container: null };

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

  function dsUrl(lang, path) {
    return new URL(CAT.DS + lang + "/" + path, window.location.origin).href;
  }

  function fetchDatasheet(path) {
    var langs = langChain(currentLang());
    var i = 0;
    function tryNext() {
      if (i >= langs.length) return Promise.reject(new Error("not found"));
      var lang = langs[i++];
      return fetch(dsUrl(lang, path)).then(function (r) {
        if (r.ok) return r.text();
        return tryNext();
      });
    }
    return tryNext();
  }

  function openPdf(path) {
    var lang = currentLang();
    var chain = langChain(lang);
    var url = dsUrl(chain[0], path);
    window.open(url, "_blank", "noopener");
  }

  function extractBodies(html) {
    var doc = new DOMParser().parseFromString(html, "text/html");
    var parts = doc.querySelectorAll(".body");
    if (!parts.length) return "<p>Datasheet content not found.</p>";
    var out = "";
    parts.forEach(function (b) { out += b.innerHTML; });
    return out;
  }

  function demoteFirstH1(fragment) {
    var wrap = document.createElement("div");
    wrap.innerHTML = fragment;
    var h1 = wrap.querySelector("h1");
    if (h1) {
      var h2 = document.createElement("h2");
      h2.innerHTML = h1.innerHTML;
      Array.prototype.forEach.call(h1.attributes, function (a) {
        h2.setAttribute(a.name, a.value);
      });
      h1.parentNode.replaceChild(h2, h1);
    }
    return wrap.innerHTML;
  }

  function isPrerendered(container) {
    return !!(container && (
      container.getAttribute("data-prerendered") === "1" ||
      (container.querySelector && !container.querySelector(".ds-loading") && container.children.length > 0)
    ));
  }

  function linkEditionCards(container, familyKey) {
    var editions = CAT.PRODUCTS[familyKey].editions;
    container.querySelectorAll(".card").forEach(function (card) {
      var nm = card.querySelector(".nm");
      if (!nm) return;
      var text = nm.textContent.trim();
      for (var i = 0; i < editions.length; i++) {
        var e = editions[i];
        if (text.indexOf(e.n) === 0 || e.n.indexOf(text.split("(")[0].trim()) === 0) {
          var href = CAT.moduleHref(familyKey, e);
          var wrap = document.createElement("a");
          wrap.href = href;
          wrap.className = "card-link";
          wrap.innerHTML = nm.innerHTML;
          nm.innerHTML = "";
          nm.appendChild(wrap);
          break;
        }
      }
    });
  }

  function applyLoaded(container, html, familyKey, updateMeta) {
    var body = extractBodies(html);
    if (familyKey && document.querySelector(".subhero h1")) {
      body = demoteFirstH1(body);
    }
    container.innerHTML = body;
    container.setAttribute("data-prerendered", "1");
    if (familyKey) linkEditionCards(container, familyKey);
    if (updateMeta) {
      var h1 = container.querySelector("h1, h2");
      if (h1 && !document.querySelector(".subhero h1")) {
        var title = h1.textContent.replace(/\s+/g, " ").trim();
        document.title = title + " | ERA One";
        var meta = document.querySelector('meta[name="description"]');
        var lead = container.querySelector(".lead");
        if (meta && lead) meta.setAttribute("content", lead.textContent.trim().slice(0, 160));
      }
    }
  }

  function loadInto(container, path, familyKey, opts) {
    opts = opts || {};
    state.path = path;
    state.familyKey = familyKey || null;
    state.container = container;
    if (!(opts.preservePrerender && isPrerendered(container))) {
      container.innerHTML = '<p class="ds-loading">' + t("ds.loading") + '</p>';
    }
    return fetchDatasheet(path)
      .then(function (html) {
        applyLoaded(container, html, familyKey, !opts.preservePrerender);
      })
      .catch(function () {
        if (!isPrerendered(container) || container.querySelector(".ds-loading")) {
          container.innerHTML = '<p class="ds-error">' + t("ds.error") + '</p>';
        }
      });
  }

  function wirePdfBtn(btn, path) {
    if (!btn || !path) return;
    btn.addEventListener("click", function () { openPdf(path); });
  }

  function renderModulesGrid(container, familyKey) {
    var p = CAT.PRODUCTS[familyKey];
    if (!p || !container) return;
    container.innerHTML = p.editions.map(function (e) {
      var href = CAT.moduleHref(familyKey, e);
      var sub = e.tagKey ? '<small data-i18n="' + e.tagKey + '"></small>' : '';
      return '<a class="mod-link" href="' + href + '"><div class="mod-meta"><b>' + e.n + "</b>" + sub + '</div><span>→</span></a>';
    }).join("");
  }

  function initPage() {
    var editionSlug = document.body.getAttribute("data-edition-slug");
    if (editionSlug) {
      var found = CAT.findEdition(editionSlug);
      var content = document.getElementById("ds-content");
      var pdfBtn = document.getElementById("ds-pdf-btn");
      var dsPath = document.body.getAttribute("data-edition-ds") || (found && found.edition.ds);
      if (!found || !dsPath) {
        if (content) content.innerHTML = '<p class="ds-error">' + t("ds.notFound") + '</p>';
        return;
      }
      wirePdfBtn(pdfBtn, dsPath);
      state.path = dsPath;
      state.familyKey = null;
      state.container = content;
      if (!isPrerendered(content)) {
        loadInto(content, dsPath, null, {});
      }
      return;
    }

    var editionRoot = document.getElementById("edition-page");
    if (editionRoot) {
      var slug = new URLSearchParams(location.search).get("id");
      if (slug) {
        location.replace("/editions/" + encodeURIComponent(slug) + ".html");
        return;
      }
      editionRoot.innerHTML = '<div class="wrap"><p class="ds-error">' + t("ds.notFound") + '</p><a href="/index.html">' + t("common.back") + '</a></div>';
      return;
    }

    var familyKey = document.body.getAttribute("data-family");
    if (familyKey && CAT.PRODUCTS[familyKey]) {
      var fam = CAT.PRODUCTS[familyKey];
      var dsContent = document.getElementById("ds-content");
      var famPdf = document.getElementById("ds-pdf-btn");
      var modsGrid = document.getElementById("ds-modules");
      wirePdfBtn(famPdf, fam.familyDs);
      if (dsContent) {
        state.path = fam.familyDs;
        state.familyKey = familyKey;
        state.container = dsContent;
        if (isPrerendered(dsContent)) {
          linkEditionCards(dsContent, familyKey);
        } else {
          loadInto(dsContent, fam.familyDs, familyKey, {});
        }
      }
      if (modsGrid) renderModulesGrid(modsGrid, familyKey);
    }
  }

  document.addEventListener("DOMContentLoaded", initPage);
  window.addEventListener("era-lang-changed", function () {
    if (state.container && state.path) {
      loadInto(state.container, state.path, state.familyKey, { preservePrerender: false });
    }
  });
})();
