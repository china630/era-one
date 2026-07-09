/* ERA Communications / ERA Office — per-user pricing calculator (product_lines). */
(function () {
  "use strict";

  function L() { return (typeof window !== "undefined" && window.ERA_LANG) ? window.ERA_LANG : null; }
  function t(k) { var lang = L(); return lang ? lang.t(k) : k; }
  function root() { return window.ERA_PRICING || null; }
  function lic() { return window.ERA_CALC_License || null; }

  function termLabel(key, fallback) {
    return L() && L().termLabel ? L().termLabel(key, fallback) : fallback || key;
  }
  function bundleTitle(key, fallback) {
    return L() && L().bundleTitle ? L().bundleTitle(key, fallback) : fallback || key;
  }
  function moduleTitle(key) {
    return L() && L().moduleTitle ? L().moduleTitle(key) : key;
  }

  function plOf(key) {
    var r = root();
    return r && r.product_lines && r.product_lines[key] ? r.product_lines[key] : null;
  }

  function moneyFmt() {
    var r = root();
    var l = L() ? L().getLang() : "en";
    var loc = (l === "en" || l === "tr") ? "en-GB" : (l === "ar" ? "ar" : "ru-RU");
    return new Intl.NumberFormat(loc, { style: "currency", currency: (r && r.currency) || "EUR", maximumFractionDigits: 0 });
  }

  function money(n) { return moneyFmt().format(Math.round(n || 0)); }

  function bundleByKey(pl, k) {
    return (pl.bundles || []).find(function (b) { return b.key === k; }) || null;
  }

  function listMultiplier() { return 1; }

  function defaultState(lineKey) {
    var pl = plOf(lineKey);
    var defBundle = lineKey === "communications" ? "comms-full" : "office-suite";
    if (pl && pl.bundles && pl.bundles.length && !pl.bundles.some(function (b) { return b.key === defBundle; })) {
      defBundle = pl.bundles[0].key;
    }
    return {
      users: 500,
      bundle: defBundle,
      manual: false,
      selected: {},
      licenseModel: "subscription",
      term: "1y",
      perpMaintYears: 1
    };
  }

  function moduleKeys(pl) { return Object.keys(pl.modules || {}); }

  function includedSet(pl) {
    var set = {};
    moduleKeys(pl).forEach(function (k) {
      var inc = pl.modules[k].included;
      if (inc) inc.forEach(function (c) { set[c] = k; });
    });
    return set;
  }

  function compute(lineKey, state) {
    var r = root();
    var pl = plOf(lineKey);
    if (!r || !pl) return null;
    var R = listMultiplier();
    var users = Math.max(0, state.users || 0);
    var bundle = state.manual ? null : bundleByKey(pl, state.bundle);
    var bundleSet = {};
    if (bundle) bundle.modules.forEach(function (k) { bundleSet[k] = true; });
    var inc = includedSet(pl);

    var lines = [];
    moduleKeys(pl).forEach(function (key) {
      if (inc[key]) return;
      var m = pl.modules[key];
      var on = !!bundleSet[key] || !!state.selected[key];
      if (!on) return;
      var inB = !!bundleSet[key];
      var disc = inB && bundle ? (bundle.discount || 0) : 0;
      var reg = m.eu_year * users * R * (1 - disc);
      lines.push({ key: key, title: moduleTitle(key), reg: reg, inB: inB, eu: m.eu_year });
    });

    var subtotal = lines.reduce(function (s, l) { return s + l.reg; }, 0);
    var Lm = lic();
    var subscription = Lm ? Lm.subscriptionTotals(subtotal, state.term || "1y") : {
      term: { key: "1y", label: "1 year", discount: 0 }, years: 1, totalYear: subtotal, totalTerm: subtotal
    };
    var perpetual = Lm ? Lm.perpetualTotals(subtotal, state.perpMaintYears) : {
      perpOnetime: subtotal * 3, perpMaintYear: subtotal * 0.2, maintYears: 1,
      perpMaintTotal: subtotal * 0.2, perpFirstYear: subtotal * 3.2
    };

    return {
      lines: lines,
      subtotal: subtotal,
      subscription: subscription,
      perpetual: perpetual,
      users: users,
      bundle: bundle,
      licenseModel: state.licenseModel || "subscription"
    };
  }

  function fillTermSelect(sel, state) {
    if (!sel || !lic()) return;
    var cur = state.term || "1y";
    sel.innerHTML = "";
    lic().termDiscounts().forEach(function (x) {
      var o = document.createElement("option");
      o.value = x.key;
      o.textContent = termLabel(x.key, x.label) + (x.discount ? " (−" + Math.round(x.discount * 100) + "%)" : "");
      sel.appendChild(o);
    });
    sel.value = cur;
  }

  function renderResult(el, res, pl, state, lineKey) {
    if (!el || !res) return;
    var Lm = lic();
    var html = "";
    if (!res.lines.length) {
      html = '<p class="pick">' + t("pickModules") + "</p>";
    } else {
      res.lines.forEach(function (l) {
        html += '<div class="line"><span class="l">' + l.title +
          (l.inB ? ' <span class="badge-disc">' + t("bundleBadge") + "</span>" : "") +
          '</span><span class="v">' + money(l.reg) + t("perYear") + "</span></div>";
      });
      if (Lm) {
        html += Lm.buildSummaryHtml({
          t: t,
          money: money,
          licenseModel: state.licenseModel,
          linesHtml: "",
          subscription: res.subscription,
          perpetual: res.perpetual,
          termLabel: termLabel
        });
      } else {
        html += '<div class="total"><span class="l">' + t("totalYear") + '</span><span class="big">' + money(res.subtotal) + "</span></div>";
      }
    }

    var mailBody = Lm ? Lm.mailQuoteBody({
      t: t,
      money: money,
      users: state.users,
      licenseModel: state.licenseModel,
      subscription: res.subscription,
      perpetual: res.perpetual,
      termLabel: termLabel(res.subscription.term.key, res.subscription.term.label)
    }) : t("mailBody");

    var productTag = lineKey === "communications" ? "ERA Communications" : "ERA Office";
    html += '<a class="cta" href="mailto:sales@era-one.solutions?subject=' +
      encodeURIComponent(t("mailSubject") + " — " + productTag) + "&body=" +
      encodeURIComponent(mailBody) + '">' + t("ctaQuote") + "</a>";

    html += '<p class="note">' + t("disclaimer") + "</p>";
    el.innerHTML = html;
  }

  function syncModules(state, pl, box) {
    var bundle = state.manual ? null : bundleByKey(pl, state.bundle);
    var bundleSet = {};
    if (bundle) bundle.modules.forEach(function (k) { bundleSet[k] = true; });
    box.querySelectorAll("[data-mod]").forEach(function (row) {
      var key = row.getAttribute("data-mod");
      var cb = row.querySelector('input[type="checkbox"]');
      if (!cb) return;
      if (!state.manual && bundle) {
        cb.checked = !!bundleSet[key];
        cb.disabled = true;
      } else {
        cb.disabled = false;
        cb.checked = !!state.selected[key];
      }
    });
  }

  function initLine(lineKey, container) {
    var pl = plOf(lineKey);
    var r = root();
    if (!pl || !r) return;

    var state = defaultState(lineKey);
    var Lm = lic();

    var html =
      '<div id="era-calc">' +
      '  <div class="calc">' +
      '    <div class="panel calc-form">' +
      '      <h3 data-i18n-calc="calcParams">' + t("calcParams") + "</h3>" +
      '      <div class="calc-grid">' +
      '        <div class="field"><label for="era-calc-users" data-i18n-calc="calcUsers">' + t("calcUsers") + '</label>' +
      '          <input type="number" id="era-calc-users" min="0" value="' + state.users + '" /></div>' +
      '        <div class="field field-full"><label for="era-calc-bundle" data-i18n-calc="calcBundle">' + t("calcBundle") + "</label>" +
      '          <select id="era-calc-bundle"></select></div>' +
      (Lm ? Lm.licenseModelFieldsHtml(t) : "") +
      "      </div>" +
      '      <h3 class="modules-title" data-i18n-calc="calcModules">' + t("calcModules") + "</h3>" +
      '      <div class="modules" data-mod-box></div>' +
      "    </div>" +
      '    <div class="panel summary"><h3 data-i18n-calc="calcSummary">' + t("calcSummary") + '</h3><div data-result></div></div>' +
      "  </div></div>";

    container.innerHTML = html;
    var rootEl = container.querySelector("#era-calc");
    var sel = rootEl.querySelector("#era-calc-bundle");
    var modBox = rootEl.querySelector("[data-mod-box]");
    var resultEl = rootEl.querySelector("[data-result]");
    var termSel = rootEl.querySelector("#term");
    var maintSel = rootEl.querySelector("#perp_maint_years");

    function fillBundleSelect() {
      sel.innerHTML =
        (pl.bundles || []).map(function (b) {
          return '<option value="' + b.key + '">' + bundleTitle(b.key, b.title) + "</option>";
        }).join("") +
        '<option value="__manual__">' + t("bundleNone") + "</option>";
      sel.value = state.manual ? "__manual__" : state.bundle;
    }

    fillBundleSelect();

    if (Lm) {
      fillTermSelect(termSel, state);
      Lm.fillMaintYearsSelect(maintSel, state, t);
    }

    modBox.innerHTML = moduleKeys(pl).map(function (key) {
      var m = pl.modules[key];
      return '<label class="mod" data-mod="' + key + '">' +
        '<input type="checkbox" />' +
        '<div class="mod-body"><span class="mname">' + moduleTitle(key) + "</span></div>" +
        '<span class="mprice">€' + m.eu_year + t("perUserYear") + "</span></label>";
    }).join("");

    function recalc() {
      syncModules(state, pl, modBox);
      renderResult(resultEl, compute(lineKey, state), pl, state, lineKey);
      if (Lm) Lm.syncLicenseModelUI(rootEl, state);
    }

    rootEl.querySelector("#era-calc-users").addEventListener("input", function (e) {
      state.users = +(e.target.value || 0);
      recalc();
    });

    if (termSel) {
      termSel.addEventListener("change", function () {
        state.term = termSel.value;
        recalc();
      });
    }

    if (Lm) {
      Lm.bindLicenseModelUI(rootEl, state, recalc);
    }

    sel.addEventListener("change", function () {
      if (sel.value === "__manual__") {
        state.manual = true;
        state.selected = {};
      } else {
        state.manual = false;
        state.bundle = sel.value;
      }
      recalc();
    });

    modBox.addEventListener("change", function (e) {
      if (e.target.type !== "checkbox") return;
      var row = e.target.closest("[data-mod]");
      if (!row) return;
      state.manual = true;
      sel.value = "__manual__";
      state.selected[row.getAttribute("data-mod")] = e.target.checked;
      recalc();
    });

    if (window.ERA_CALC_applyStatic) window.ERA_CALC_applyStatic();
    recalc();

    document.addEventListener("eralangchange", function () {
      if (window.ERA_CALC_applyStatic) window.ERA_CALC_applyStatic();
      fillBundleSelect();
      if (Lm) {
        fillTermSelect(termSel, state);
        Lm.fillMaintYearsSelect(maintSel, state, t);
      }
      modBox.querySelectorAll("[data-mod]").forEach(function (row) {
        var key = row.getAttribute("data-mod");
        var name = row.querySelector(".mname");
        if (name) name.textContent = moduleTitle(key);
      });
      recalc();
    });
  }

  window.ERA_mountProductCalc = function (container, lineKey) {
    if (!container || !lineKey) return;
    initLine(lineKey, container);
  };

  window.ERA_computeProductLine = compute;
})();
