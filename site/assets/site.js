/* ERA One site — shared chrome, i18n, search, calculators. Catalog: products-catalog.js */
(function () {
  var I18N = window.ERA_I18N || {};
  var CAT = window.ERA_CATALOG;
  var LABEL = { en: "EN", ru: "RU", tr: "TR", ar: "AR" };

  function t(dict, k) { return (dict && dict[k] != null) ? dict[k] : (I18N.en[k] != null ? I18N.en[k] : k); }

  function products() { return CAT ? CAT.PRODUCTS : {}; }
  function fams() { return CAT ? CAT.FAMS : []; }
  function moduleHref(fk, e) { return CAT ? CAT.moduleHref(fk, e) : "#"; }

  /* ---- Search index ---- */
  function buildSearchIndex() {
    if (!CAT) return [];
    var idx = CAT.FAMS.map(function (f) { return { n: f.name, g: "Product", h: f.page }; });
    CAT.FAMS.forEach(function (f) {
      CAT.PRODUCTS[f.key].editions.forEach(function (e) {
        idx.push({ n: e.n, g: f.name, h: moduleHref(f.key, e) });
      });
    });
    return idx;
  }
  var SEARCH_INDEX = buildSearchIndex();

  /* ---- Mega menu ---- */
  function megaHTML() {
    if (!CAT) return "";
    var famList = CAT.FAMS.map(function (f, i) {
      return '<a class="fam' + (i === 0 ? ' active' : '') + '" data-fam="' + f.key + '" href="' + f.page + '">' +
        '<b>' + f.name + '</b><span data-i18n="' + f.tagKey + '"></span></a>';
    }).join("");
    var modPanels = CAT.FAMS.map(function (f) {
      var list = CAT.PRODUCTS[f.key].editions.map(function (e) {
        var tag = e.tagKey ? '<span class="mod-tag" data-i18n="' + e.tagKey + '"></span>' : '';
        return '<a href="' + moduleHref(f.key, e) + '">' + e.n + tag + '</a>';
      }).join("");
      return '<div class="mods" data-fam="' + f.key + '"' + (f.key !== "control" ? ' hidden' : '') + '>' +
        '<div class="mods-head"><a href="' + f.page + '"><b>' + f.name + '</b> · <span data-i18n="common.learn">Learn more</span> →</a></div>' +
        '<div class="mods-grid">' + list + '</div>' +
        '<div class="mods-foot">' +
        '<a href="compare.html?family=' + f.key + '" data-i18n="nav.compare">Compare</a>' +
        '<a href="downloads.html" data-i18n="nav.downloads">Downloads</a>' +
        '</div></div>';
    }).join("");
    return '<div class="mega"><div class="mega-inner">' +
      '<div class="mega-fams"><div class="mega-lbl" data-i18n="nav.products">Products</div>' + famList + '</div>' +
      '<div class="mega-mods">' + modPanels + '</div></div></div>';
  }

  function footerProductCols() {
    if (!CAT) return "";
    return CAT.FAMS.map(function (f) {
      var links = '<li><a href="compare.html?family=' + f.key + '" data-i18n="nav.compare">Compare</a></li>';
      links += CAT.PRODUCTS[f.key].editions.map(function (e) {
        return '<li><a href="' + moduleHref(f.key, e) + '">' + e.n + '</a></li>';
      }).join("");
      return '<div><h4><a href="' + f.page + '">' + f.name + '</a></h4><ul>' + links + '</ul></div>';
    }).join("");
  }

  function headerHTML() {
    return '' +
      '<div class="wrap">' +
      '  <a class="brand" href="index.html"><img src="assets/era-one-logo.png" alt="ERA One" onerror="this.onerror=null;this.src=\'assets/era-one-logo.svg\'" /></a>' +
      '  <nav>' +
      '    <div class="nav-item has-mega" data-menu="products">' +
      '      <a href="index.html#products" class="nav-link" data-nav="products"><span data-i18n="nav.products">Products</span> <span class="caret">▾</span></a>' +
      megaHTML() +
      '    </div>' +
      '    <a href="index.html#why" class="nav-link" data-nav="why" data-i18n="nav.why">Why ERA One</a>' +
      '    <div class="nav-item has-drop" data-menu="company">' +
      '      <a href="about.html" class="nav-link" data-nav="company"><span data-i18n="nav.company">Company</span> <span class="caret">▾</span></a>' +
      '      <div class="drop">' +
      '        <a href="about.html" data-i18n="nav.about">About</a>' +
      '        <a href="vision.html" data-i18n="nav.vision">Vision</a>' +
      '        <a href="contacts.html" data-i18n="nav.contacts">Contacts</a>' +
      '        <a href="compare.html" data-i18n="nav.compare">Compare</a>' +
      '        <a href="downloads.html" data-i18n="nav.downloads">Downloads</a>' +
      '        <a href="contacts.html" data-i18n="nav.partners">Partners</a>' +
      '        <a href="contacts.html" data-i18n="nav.careers">Careers</a>' +
      '      </div>' +
      '    </div>' +
      '  </nav>' +
      '  <div class="search" id="search">' +
      '    <button class="icon-btn" id="searchBtn" aria-label="Search" aria-expanded="false">' +
      '      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9"><circle cx="11" cy="11" r="7"/><path d="M21 21l-4.3-4.3"/></svg>' +
      '    </button>' +
      '    <div class="search-panel" id="searchPanel">' +
      '      <input type="text" id="searchInput" data-i18n-ph="search.placeholder" placeholder="Search…" autocomplete="off" />' +
      '      <div class="search-results" id="searchResults"></div>' +
      '    </div>' +
      '  </div>' +
      '  <div class="lang">' +
      '    <button class="lang-btn" id="langBtn" aria-haspopup="true" aria-expanded="false">' +
      '      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.5 2.6 2.5 15.4 0 18M12 3c-2.5 2.6-2.5 15.4 0 18"/></svg>' +
      '      <span id="langLabel">EN</span>' +
      '    </button>' +
      '    <div class="lang-menu" id="langMenu" role="menu">' +
      '      <button data-lang="en" class="active"><span class="flag">🇬🇧</span> English</button>' +
      '      <button data-lang="ru"><span class="flag">🇷🇺</span> Русский</button>' +
      '      <button data-lang="tr"><span class="flag">🇹🇷</span> Türkçe</button>' +
      '      <button data-lang="ar"><span class="flag">🇸🇦</span> العربية</button>' +
      '    </div>' +
      '  </div>' +
      '  <a class="login-btn" href="login.html"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/><path d="M10 17l5-5-5-5M15 12H3"/></svg><span data-i18n="nav.login">Log in</span></a>' +
      '  <button class="cta-btn" data-i18n="nav.demo" onclick="location.href=\'contacts.html\'">Book a demo</button>' +
      '</div>';
  }

  function footerHTML() {
    return '' +
      '<div class="wrap">' +
      '  <div class="ftr-cols">' +
      '    <div>' +
      '      <h4 data-i18n="foot.about">Company</h4>' +
      '      <ul>' +
      '        <li><a href="about.html" data-i18n="nav.about">About</a></li>' +
      '        <li><a href="vision.html" data-i18n="nav.vision">Vision</a></li>' +
      '        <li><a href="contacts.html" data-i18n="nav.contacts">Contacts</a></li>' +
      '        <li><a href="compare.html?family=control" data-i18n="nav.compare">Compare</a></li>' +
      '        <li><a href="downloads.html" data-i18n="nav.downloads">Downloads</a></li>' +
      '        <li><a href="contacts.html" data-i18n="nav.partners">Partners</a></li>' +
      '        <li><a href="contacts.html" data-i18n="nav.careers">Careers</a></li>' +
      '      </ul>' +
      '    </div>' +
      footerProductCols() +
      '    <div class="ftr-brand">' +
      '      <img src="assets/era-one-logo.png" alt="ERA One" onerror="this.onerror=null;this.src=\'assets/era-one-logo.svg\'" />' +
      '      <div class="ct">' +
      '        <span data-i18n="foot.tagline">Sovereign IT &amp; security ecosystem for the isolated contour.</span><br />' +
      '        <a href="mailto:sales@era-one.solutions">sales@era-one.solutions</a><br />' +
      '        <a href="https://www.era-one.solutions">www.era-one.solutions</a>' +
      '      </div>' +
      '    </div>' +
      '  </div>' +
      '  <div class="ftr-bottom">' +
      '    <span data-i18n="foot.rights">© 2026 ERA One. All rights reserved.</span>' +
      '    <span data-i18n="foot.fine">Indicative product information — not a public offer.</span>' +
      '  </div>' +
      '</div>';
  }

  /* ---- Calculator ---- */
  function bundlePerUser(productLine, bundleKey) {
    var root = window.ERA_PRICING;
    if (!root || !root.product_lines || !root.product_lines[productLine]) return null;
    var pl = root.product_lines[productLine];
    var mods = pl.modules || {};
    var bundle = (pl.bundles || []).find(function (b) { return b.key === bundleKey; });
    if (!bundle) return null;
    var sum = 0;
    bundle.modules.forEach(function (k) {
      if (mods[k] && mods[k].eu_year != null) sum += mods[k].eu_year;
    });
    var disc = bundle.discount || 0;
    var cis = (root.regions && root.regions.cis && root.regions.cis.multiplier) || 0.5;
    return sum * (1 - disc) * cis;
  }

  function renderCalc(container, productKey, dict) {
    var p = products()[productKey];
    if (!p) return;
    var model = p.calc;
    if (model === "control" && typeof window.ERA_mountControlCalc === "function") {
      window.ERA_mountControlCalc(container);
      return;
    }
    var html = '<div class="calc">';
    html += '<div class="section-head" style="margin-bottom:6px"><div class="eyebrow" data-i18n="calc.h2">' + t(dict, "calc.h2") + '</div>' +
      '<p data-i18n="calc.sub" style="margin-top:6px">' + t(dict, "calc.sub") + '</p></div>';
    if (model === "control") {
      html += '<div class="row"><label data-i18n="calc.ws">Workstations</label><input type="number" min="0" value="100" id="c_ws"></div>';
      html += '<div class="row"><label data-i18n="calc.servers">Servers (×3)</label><input type="number" min="0" value="10" id="c_srv"></div>';
    } else {
      html += '<div class="row"><label data-i18n="calc.users">Users</label><input type="number" min="0" value="100" id="c_users"></div>';
    }
    html += '<div class="result"><span class="lbl" data-i18n="calc.result">Indicative / year</span><span class="val" id="c_val">—</span></div>';
    html += '<div class="note" data-i18n="calc.note">' + t(dict, "calc.note") + '</div>';
    html += '<div class="calc-cta"><a class="btn-sm" href="contacts.html" data-i18n="calc.cta">' + t(dict, "calc.cta") + '</a></div>';
    html += '</div>';
    container.innerHTML = html;
    function recalc() {
      var total = 0, cur = "€";
      if (model === "control") {
        var ws = +(document.getElementById("c_ws").value || 0);
        var srv = +(document.getElementById("c_srv").value || 0);
        total = ws * 12 + srv * 36;
      } else {
        var u = +(document.getElementById("c_users").value || 0);
        var rate = null;
        if (productKey === "communications") rate = bundlePerUser("communications", "comms-full");
        if (productKey === "office") rate = bundlePerUser("office", "office-suite");
        if (rate == null) rate = 14;
        total = u * rate;
      }
      document.getElementById("c_val").textContent = "€ " + Math.round(total).toLocaleString("en-US");
    }
    container.querySelectorAll("input").forEach(function (i) { i.addEventListener("input", recalc); });
    recalc();
  }

  function applyLang(lang) {
    var dict = I18N[lang] || I18N.en;
    document.documentElement.lang = lang;
    document.documentElement.dir = (lang === "ar") ? "rtl" : "ltr";
    var ll = document.getElementById("langLabel");
    if (ll) ll.textContent = LABEL[lang];
    document.querySelectorAll("[data-i18n]").forEach(function (el) {
      var k = el.getAttribute("data-i18n"); var v = t(dict, k);
      if (v != null) el.innerHTML = v;
    });
    document.querySelectorAll("[data-i18n-ph]").forEach(function (el) {
      var k = el.getAttribute("data-i18n-ph"); var v = t(dict, k);
      if (v != null) el.setAttribute("placeholder", v);
    });
    document.querySelectorAll(".lang-menu button").forEach(function (b) {
      b.classList.toggle("active", b.dataset.lang === lang);
    });
    try { localStorage.setItem("era-lang", lang); } catch (e) {}
    window.dispatchEvent(new CustomEvent("era-lang-changed", { detail: { lang: lang } }));
  }
  window.eraApplyLang = applyLang;

  document.addEventListener("DOMContentLoaded", function () {
    SEARCH_INDEX = buildSearchIndex();
    var head = document.getElementById("site-header");
    var foot = document.getElementById("site-footer");
    if (head) head.innerHTML = headerHTML();
    if (foot) foot.innerHTML = footerHTML();

    var page = document.body.getAttribute("data-page");
    if (page) {
      var link = document.querySelector('[data-nav="' + page + '"]');
      if (link) link.classList.add("active");
    }

    var lang = "en";
    try { lang = localStorage.getItem("era-lang") || "en"; } catch (e) {}

    document.querySelectorAll("[data-calc]").forEach(function (c) {
      renderCalc(c, c.getAttribute("data-calc"), I18N[lang] || I18N.en);
    });

    applyLang(lang);

    var lbtn = document.getElementById("langBtn");
    var lmenu = document.getElementById("langMenu");
    if (lbtn && lmenu) {
      lbtn.addEventListener("click", function (e) {
        e.stopPropagation();
        lmenu.classList.toggle("open");
        lbtn.setAttribute("aria-expanded", lmenu.classList.contains("open"));
      });
      lmenu.querySelectorAll("button").forEach(function (b) {
        b.addEventListener("click", function () { applyLang(b.dataset.lang); lmenu.classList.remove("open"); });
      });
      document.addEventListener("click", function () { lmenu.classList.remove("open"); });
    }

    document.querySelectorAll(".mega .fam").forEach(function (fam) {
      fam.addEventListener("mouseenter", function () {
        var key = fam.getAttribute("data-fam");
        document.querySelectorAll(".mega .fam").forEach(function (f) { f.classList.toggle("active", f === fam); });
        document.querySelectorAll(".mega .mods").forEach(function (m) { m.hidden = (m.getAttribute("data-fam") !== key); });
      });
    });

    var sBtn = document.getElementById("searchBtn");
    var sPanel = document.getElementById("searchPanel");
    var sInput = document.getElementById("searchInput");
    var sRes = document.getElementById("searchResults");
    function renderResults(q) {
      var dict = I18N[document.documentElement.lang] || I18N.en;
      q = (q || "").trim().toLowerCase();
      var items = q ? SEARCH_INDEX.filter(function (i) { return i.n.toLowerCase().indexOf(q) !== -1; }) : SEARCH_INDEX;
      if (!items.length) { sRes.innerHTML = '<div class="empty">' + t(dict, "search.empty") + '</div>'; return; }
      sRes.innerHTML = items.map(function (i) {
        return '<a href="' + i.h + '"><span>' + i.n + '</span><small>' + i.g + '</small></a>';
      }).join("");
    }
    if (sBtn && sPanel) {
      sBtn.addEventListener("click", function (e) {
        e.stopPropagation();
        var open = sPanel.classList.toggle("open");
        sBtn.setAttribute("aria-expanded", open);
        if (open) { renderResults(""); sInput.value = ""; sInput.focus(); }
      });
      sPanel.addEventListener("click", function (e) { e.stopPropagation(); });
      sInput.addEventListener("input", function () { renderResults(sInput.value); });
      document.addEventListener("click", function () { sPanel.classList.remove("open"); });
    }
  });
})();
