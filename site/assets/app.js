/* ERA One — калькулятор цен (ADR-0021). Данные: window.ERA_PRICING (из pricing-data.yaml). */
(function () {
  "use strict";
  var G = typeof window !== "undefined" ? window : (typeof global !== "undefined" ? global : {});
  var D = G.ERA_PRICING;
  if (!D) { if (typeof console !== "undefined") console.error("ERA_PRICING не загружен"); return; }

  var SPECIAL_UNITS = { device: 1, source: 1, technician: 1, node: 1, admin: 1, site: 1, hub: 1 };
  var moneyFmt = null;

  var state = {
    region: "cis",
    ws: 1000,
    servers: 50,
    term: "3y_prepaid",
    licenseModel: "subscription",
    perpMaintYears: 1,
    bundle: "secops",
    selected: {},
    qty: {}
  };

  function L() { return (typeof window !== "undefined" && window.ERA_LANG) ? window.ERA_LANG : null; }
  function t(k) { var lang = L(); return lang ? lang.t(k) : k; }
  function unitLabel(u) { return t("unit." + u) || u; }
  function termLabel(key, fallback) {
    var v = t("termLabel." + key);
    return (v && v.indexOf("termLabel.") !== 0) ? v : fallback;
  }
  function bundleTitle(key, fallback) {
    var v = t("bundleTitle." + key);
    return (v && v.indexOf("bundleTitle.") !== 0) ? v : fallback;
  }

  function fmtMoney() {
    var l = L() ? L().getLang() : "en";
    var loc = (l === "en" || l === "tr") ? "en-GB" : (l === "ar" ? "ar" : "ru-RU");
    moneyFmt = new Intl.NumberFormat(loc, { style: "currency", currency: D.currency || "EUR", maximumFractionDigits: 0 });
  }
  function money(n) {
    if (!moneyFmt) fmtMoney();
    return moneyFmt.format(Math.round(n || 0));
  }

  function moduleKeys() { return Object.keys(D.modules); }

  function volumeFor(count) {
    for (var i = 0; i < D.volume_discounts.length; i++) {
      var tier = D.volume_discounts[i];
      var okMin = count >= tier.min;
      var okMax = (tier.max === null || tier.max === undefined) ? true : count <= tier.max;
      if (okMin && okMax) {
        return { discount: tier.discount, byRequest: (tier.discount === null || tier.discount === undefined) };
      }
    }
    return { discount: 0, byRequest: false };
  }

  function flatFor(count) {
    var m = D.modules["control-ai"];
    if (!m || !m.flat_alt) return null;
    for (var i = 0; i < m.flat_alt.length; i++) {
      var tier = m.flat_alt[i];
      if (tier.up_to === null || tier.up_to === undefined) return null;
      if (count <= tier.up_to) return tier.eu_year;
    }
    return null;
  }

  function termByKey(k) {
    return D.term_discounts.find(function (x) { return x.key === k; }) || D.term_discounts[0];
  }
  function bundleByKey(k) {
    return D.bundles.find(function (b) { return b.key === k; }) || null;
  }
  function yearsOf(termKey) {
    if (termKey === "5y_prepaid") return 5;
    if (termKey.indexOf("3y") === 0) return 3;
    return 1;
  }

  function compute() {
    var R = D.regions[state.region].multiplier;
    var S = D.server_multiplier || 3;
    var epCount = (state.ws || 0) + (state.servers || 0);
    var epWeighted = (state.ws || 0) + (state.servers || 0) * S;
    var bundle = bundleByKey(state.bundle);
    var bundleSet = {};
    if (bundle) bundle.modules.forEach(function (k) { bundleSet[k] = true; });
    var vol = volumeFor(epCount);
    var term = termByKey(state.term);

    var lines = [];
    moduleKeys().forEach(function (key) {
      var m = D.modules[key];
      var isReq = !!m.required;
      if (!isReq && !state.selected[key]) return;

      var qty, euBase, isEndpoint = (m.unit === "endpoint");
      if (isEndpoint) {
        qty = epWeighted;
        euBase = m.eu_year * qty;
        if (key === "control-ai" && m.flat_alt) {
          var flat = flatFor(epCount);
          if (flat !== null && flat < euBase) euBase = flat;
        }
      } else {
        qty = state.qty[key] || 0;
        euBase = m.eu_year * qty;
      }
      if (!isEndpoint && qty <= 0) return;

      var inB = !!bundleSet[key];
      var bDisc = inB && bundle ? bundle.discount : 0;
      var reg = euBase * R * (1 - bDisc);
      if (isEndpoint && vol.discount) reg *= (1 - vol.discount);

      lines.push({ key: key, title: m.title, unit: m.unit, qty: qty, reg: reg, inB: inB, isEndpoint: isEndpoint });

      if (key === "pam" && m.addon && (state.qty.pam_target || 0) > 0) {
        var ad = m.addon[0];
        var tQty = state.qty.pam_target;
        var tReg = ad.eu_year * tQty * R * (1 - bDisc);
        lines.push({ key: "pam_target", title: ad.title, unit: ad.unit, qty: tQty, reg: tReg, inB: inB, isEndpoint: false });
      }
    });

    var subtotal = lines.reduce(function (s, l) { return s + l.reg; }, 0);
    var lic = G.ERA_CALC_License;
    var sub = lic ? lic.subscriptionTotals(subtotal, state.term) : {
      term: term, years: yearsOf(term.key),
      totalYear: subtotal * (1 - term.discount), totalTerm: subtotal * (1 - term.discount) * yearsOf(term.key)
    };
    var perp = lic ? lic.perpetualTotals(subtotal, state.perpMaintYears) : {
      perpOnetime: subtotal * 3, perpMaintYear: subtotal * 0.2, maintYears: 1,
      perpMaintTotal: subtotal * 0.2, perpFirstYear: subtotal * 3.2
    };
    return {
      lines: lines, subtotal: subtotal, vol: vol, term: term, years: sub.years,
      subscription: sub, perpetual: perp,
      totalYear: sub.totalYear, totalTerm: sub.totalTerm,
      perpOnetime: perp.perpOnetime, perpMaint: perp.perpMaintYear,
      epCount: epCount, region: state.region, licenseModel: state.licenseModel
    };
  }

  function defaultQty(key) {
    var m = D.modules[key];
    if (!m) return 0;
    if (m.unit === "technician") return 3;
    if (m.unit === "admin") return 5;
    if (m.unit === "site" || m.unit === "hub") return 1;
    return 10;
  }

  function toggleQty(key) {
    var box = document.getElementById("qtybox_" + key);
    if (!box) return;
    var on = !!state.selected[key] || !!D.modules[key].required;
    box.style.display = on ? "grid" : "none";
    if (on && !state.qty[key] && SPECIAL_UNITS[D.modules[key].unit]) {
      state.qty[key] = defaultQty(key);
      var i = document.getElementById("qty_" + key);
      if (i) i.value = state.qty[key];
    }
  }

  function buildModuleRow(key) {
    var m = D.modules[key];
    var isReq = !!m.required;
    var row = document.createElement("div");
    row.className = "mod";
    row.dataset.key = key;

    var cb = document.createElement("input");
    cb.type = "checkbox";
    cb.id = "chk_" + key;
    cb.checked = isReq || !!state.selected[key];
    cb.disabled = isReq;
    cb.onchange = function () {
      state.selected[key] = cb.checked;
      toggleQty(key);
      render();
    };

    var body = document.createElement("div");
    body.className = "mod-body";
    var tag = m.availability === "project" ? ' <span class="tag">' + t("projectTag") + "</span>" : "";
    body.innerHTML = '<div class="mname">' + m.title + tag + '</div><div class="mmeta" id="desc_' + key + '"></div>';
    body.querySelector("#desc_" + key).textContent = L() ? L().moduleDesc(key) : (m.desc || "");

    var price = document.createElement("div");
    price.className = "mprice";
    price.id = "price_" + key;

    row.appendChild(cb);
    row.appendChild(body);
    row.appendChild(price);

    if (SPECIAL_UNITS[m.unit]) {
      var q = document.createElement("div");
      q.className = "qty";
      q.id = "qtybox_" + key;

      var f1 = document.createElement("div");
      var lab1 = document.createElement("label");
      lab1.setAttribute("for", "qty_" + key);
      lab1.textContent = t("qty") + " (" + unitLabel(m.unit) + ")";
      var inp = document.createElement("input");
      inp.type = "number";
      inp.min = "0";
      inp.id = "qty_" + key;
      inp.value = state.qty[key] || "";
      inp.oninput = function () {
        state.qty[key] = parseInt(inp.value || "0", 10) || 0;
        render();
      };
      f1.appendChild(lab1);
      f1.appendChild(inp);
      q.appendChild(f1);

      if (key === "pam" && m.addon) {
        var f2 = document.createElement("div");
        var lab2 = document.createElement("label");
        lab2.setAttribute("for", "qty_pam_target");
        lab2.textContent = t("pamTargets");
        var inp2 = document.createElement("input");
        inp2.type = "number";
        inp2.min = "0";
        inp2.id = "qty_pam_target";
        inp2.value = state.qty.pam_target || "";
        inp2.oninput = function () {
          state.qty.pam_target = parseInt(inp2.value || "0", 10) || 0;
          render();
        };
        f2.appendChild(lab2);
        f2.appendChild(inp2);
        q.appendChild(f2);
      }
      row.appendChild(q);
    }
    return row;
  }

  function buildModules() {
    var host = document.getElementById("modules");
    if (!host) return;
    var openState = {};
    host.querySelectorAll(".mod-group").forEach(function (d) {
      openState[d.dataset.group] = d.open;
    });
    host.innerHTML = "";
    var groups = window.ERA_MODULE_GROUPS || [{ key: "all", modules: moduleKeys() }];
    groups.forEach(function (grp) {
      var details = document.createElement("details");
      details.className = "mod-group";
      details.dataset.group = grp.key;
      details.open = openState[grp.key] === true;

      var summary = document.createElement("summary");
      summary.className = "mod-group-toggle";
      summary.textContent = L() ? L().catLabel(grp.key) : grp.key;
      details.appendChild(summary);

      var body = document.createElement("div");
      body.className = "mod-group-body";
      grp.modules.forEach(function (key) {
        if (!D.modules[key]) return;
        body.appendChild(buildModuleRow(key));
      });
      details.appendChild(body);
      host.appendChild(details);
    });
    syncModuleUI();
  }

  function fillSelects() {
    var term = document.getElementById("term");
    var bundle = document.getElementById("bundle");
    if (!term || !bundle) return;

    var termVal = term.value || state.term;
    var bundleVal = bundle.value || state.bundle;

    term.innerHTML = "";
    D.term_discounts.forEach(function (x) {
      var o = document.createElement("option");
      o.value = x.key;
      o.textContent = termLabel(x.key, x.label) + (x.discount ? " (−" + Math.round(x.discount * 100) + "%)" : "");
      term.appendChild(o);
    });
    term.value = termVal;
    term.onchange = function () { state.term = term.value; render(); };

    bundle.innerHTML = "";
    var none = document.createElement("option");
    none.value = "";
    none.textContent = t("bundleNone");
    bundle.appendChild(none);
    D.bundles.forEach(function (b) {
      var o = document.createElement("option");
      o.value = b.key;
      o.textContent = bundleTitle(b.key, b.title) + " (−" + Math.round(b.discount * 100) + "%)";
      bundle.appendChild(o);
    });
    bundle.value = bundleVal;
    bundle.onchange = function () {
      state.bundle = bundle.value;
      var bd = bundleByKey(state.bundle);
      if (bd) bd.modules.forEach(function (k) {
        state.selected[k] = true;
        var mm = D.modules[k];
        if (mm && SPECIAL_UNITS[mm.unit] && !state.qty[k]) state.qty[k] = defaultQty(k);
      });
      syncModuleUI();
      render();
    };
  }

  function buildControls() {
    document.querySelectorAll("#region .seg button").forEach(function (b) {
      b.classList.toggle("active", b.dataset.region === state.region);
      b.onclick = function () {
        state.region = b.dataset.region;
        syncRegionUI();
        render();
      };
    });

    var ws = document.getElementById("ws");
    var sv = document.getElementById("servers");
    if (ws) { ws.value = state.ws; ws.oninput = function () { state.ws = parseInt(ws.value || "0", 10) || 0; render(); }; }
    if (sv) { sv.value = state.servers; sv.oninput = function () { state.servers = parseInt(sv.value || "0", 10) || 0; render(); }; }

    fillSelects();
    buildModules();
    syncRegionUI();
    var eraRoot = document.getElementById("era-calc");
    if (eraRoot && G.ERA_CALC_License) {
      G.ERA_CALC_License.fillMaintYearsSelect(
        document.getElementById("perp_maint_years"), state, t
      );
      G.ERA_CALC_License.bindLicenseModelUI(eraRoot, state, render);
    }
  }

  function syncModuleUI() {
    moduleKeys().forEach(function (key) {
      var m = D.modules[key];
      var cb = document.getElementById("chk_" + key);
      if (cb) cb.checked = !!m.required || !!state.selected[key];
      var desc = document.getElementById("desc_" + key);
      if (desc && L()) desc.textContent = L().moduleDesc(key);
      if (SPECIAL_UNITS[m.unit]) toggleQty(key);
    });
  }

  function syncRegionUI() {
    document.querySelectorAll("#region .seg button").forEach(function (b) {
      b.classList.toggle("active", b.dataset.region === state.region);
    });
    var note = document.getElementById("region-note");
    if (note) note.style.display = state.region === "cis" ? "block" : "none";

    var R = D.regions[state.region].multiplier;
    moduleKeys().forEach(function (key) {
      var m = D.modules[key];
      var el = document.getElementById("price_" + key);
      if (!el) return;
      var p = m.eu_year * R;
      el.textContent = p === 0 ? t("included") : (money(p) + "/" + unitLabel(m.unit) + t("perYear"));
    });
  }

  function render() {
    syncRegionUI();
    var r = compute();
    var out = document.getElementById("result");
    if (!out) return;
    var html = "";
    var lic = G.ERA_CALC_License;

    r.lines.forEach(function (l) {
      var u = unitLabel(l.unit);
      var qtyTxt = l.isEndpoint ? (l.qty + " (" + t("weighted") + ")") : (l.qty + " " + u);
      html += '<div class="line"><span class="l">' + l.title +
        ' <span style="color:#8aa0ab">× ' + qtyTxt + "</span>" +
        (l.inB ? ' <span class="badge-disc">' + t("bundleBadge") + "</span>" : "") +
        '</span><span class="v">' + money(l.reg) + t("perYear") + "</span></div>";
    });
    if (r.lines.length === 0) {
      html += '<div class="line"><span class="l">' + t("pickModules") + '</span><span class="v">—</span></div>';
    }

    if (lic) {
      html += lic.buildSummaryHtml({
        t: t,
        money: money,
        licenseModel: state.licenseModel,
        linesHtml: "",
        subscription: r.subscription,
        perpetual: r.perpetual,
        termLabel: termLabel,
        volByRequest: r.vol.byRequest,
        volDiscount: !r.vol.byRequest && r.vol.discount ? r.vol.discount : 0,
        epCount: r.epCount
      });
    }

    var mailBody = lic ? lic.mailQuoteBody({
      t: t,
      money: money,
      regionLabel: D.regions[state.region].label,
      ws: state.ws,
      servers: state.servers,
      licenseModel: state.licenseModel,
      subscription: r.subscription,
      perpetual: r.perpetual,
      termLabel: termLabel(r.subscription.term.key, r.subscription.term.label)
    }) : (t("mailBody") + "\n" + money(r.totalYear));

    html += '<a class="cta" href="mailto:sales@era-one.solutions?subject=' +
      encodeURIComponent(t("mailSubject")) + "&body=" + encodeURIComponent(mailBody) +
      '">' + t("ctaQuote") + "</a>";

    var disc = (L() && L().t("disclaimer")) || D.disclaimer;
    html += '<div class="note">' + disc + "</div>";
    out.innerHTML = html;
    if (lic) lic.syncLicenseModelUI(document.getElementById("era-calc"), state);
  }

  function onLangChange() {
    fmtMoney();
    fillSelects();
    if (G.ERA_CALC_License) {
      G.ERA_CALC_License.fillMaintYearsSelect(document.getElementById("perp_maint_years"), state, t);
    }
    buildModules();
    render();
  }

  function init() {
    var bd = bundleByKey("secops");
    if (bd) bd.modules.forEach(function (k) { state.selected[k] = true; });
    buildControls();
    var bundleEl = document.getElementById("bundle");
    if (bundleEl) bundleEl.value = state.bundle;
    render();
  }

  function boot() {
    if (!document.getElementById("modules")) return;
    init();
  }

  if (typeof document !== "undefined") {
    document.addEventListener("DOMContentLoaded", boot);
    document.addEventListener("eralangchange", onLangChange);
    window.ERA_CALC_init = init;
  }

  if (typeof module !== "undefined" && module.exports) {
    module.exports = {
      compute: compute,
      state: state,
      setState: function (s) { for (var k in s) state[k] = s[k]; },
      volumeFor: volumeFor,
      flatFor: flatFor
    };
  }
})();
