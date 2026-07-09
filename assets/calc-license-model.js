/* Shared Subscription / Perpetual logic for ERA Control, Office, Communications calculators. */
(function () {
  "use strict";

  function root() { return window.ERA_PRICING || {}; }

  function perpCfg() {
    var r = root();
    return r.perpetual || { multiplier_of_annual: 3, maintenance_rate: 0.2 };
  }

  function termDiscounts() {
    var r = root();
    return r.term_discounts || [
      { key: "1y", label: "1 year", discount: 0 },
      { key: "3y_annual", label: "3 years, annual", discount: 0.1 },
      { key: "3y_prepaid", label: "3 years, prepaid", discount: 0.2 },
      { key: "5y_prepaid", label: "5 years, prepaid", discount: 0.25 }
    ];
  }

  function termByKey(k) {
    return termDiscounts().find(function (x) { return x.key === k; }) || termDiscounts()[0];
  }

  function yearsOf(termKey) {
    if (termKey === "5y_prepaid") return 5;
    if (termKey && termKey.indexOf("3y") === 0) return 3;
    return 1;
  }

  function subscriptionTotals(subtotal, termKey) {
    var term = termByKey(termKey);
    var totalYear = subtotal * (1 - (term.discount || 0));
    var years = yearsOf(term.key);
    return {
      subtotal: subtotal,
      term: term,
      years: years,
      totalYear: totalYear,
      totalTerm: totalYear * years
    };
  }

  function perpetualTotals(subtotal, maintYears) {
    var p = perpCfg();
    var mult = p.multiplier_of_annual || 3;
    var rate = p.maintenance_rate || 0.2;
    var my = Math.max(1, parseInt(maintYears, 10) || 1);
    var perpMaintYear = subtotal * rate;
    return {
      subtotal: subtotal,
      perpOnetime: subtotal * mult,
      perpMaintYear: perpMaintYear,
      maintYears: my,
      perpMaintTotal: perpMaintYear * my,
      perpFirstYear: subtotal * mult + perpMaintYear
    };
  }

  function syncLicenseModelUI(container, state) {
    if (!container) return;
    var model = state.licenseModel || "subscription";
    container.querySelectorAll("[data-license-model-seg] button").forEach(function (b) {
      b.classList.toggle("active", b.getAttribute("data-license-model") === model);
    });
    container.querySelectorAll("[data-sub-only]").forEach(function (el) {
      el.style.display = model === "subscription" ? "" : "none";
    });
    container.querySelectorAll("[data-perp-only]").forEach(function (el) {
      el.style.display = model === "perpetual" ? "" : "none";
    });
  }

  function bindLicenseModelUI(container, state, onChange) {
    if (!container) return;
    var seg = container.querySelector("[data-license-model-seg]");
    if (!seg) return;
    seg.addEventListener("click", function (e) {
      var btn = e.target.closest("[data-license-model]");
      if (!btn) return;
      state.licenseModel = btn.getAttribute("data-license-model");
      syncLicenseModelUI(container, state);
      if (onChange) onChange();
    });
    var maintSel = container.querySelector("#perp_maint_years");
    if (maintSel) {
      maintSel.addEventListener("change", function () {
        state.perpMaintYears = parseInt(maintSel.value, 10) || 1;
        if (onChange) onChange();
      });
    }
    syncLicenseModelUI(container, state);
  }

  function fillMaintYearsSelect(sel, state, t) {
    if (!sel) return;
    var cur = String(state.perpMaintYears || 1);
    sel.innerHTML =
      '<option value="1">' + t("perpMaint1y") + "</option>" +
      '<option value="3">' + t("perpMaint3y") + "</option>" +
      '<option value="5">' + t("perpMaint5y") + "</option>";
    sel.value = cur;
  }

  function licenseModelFieldsHtml(t) {
    return (
      '<div class="field field-full calc-license-row">' +
      '  <div class="calc-license-model">' +
      '    <label data-i18n-calc="calcLicenseModel">' + t("calcLicenseModel") + "</label>" +
      '    <div class="seg" data-license-model-seg>' +
      '      <button type="button" data-license-model="subscription" class="active" data-i18n-calc="licenseSubscription">' + t("licenseSubscription") + "</button>" +
      '      <button type="button" data-license-model="perpetual" data-i18n-calc="licensePerpetual">' + t("licensePerpetual") + "</button>" +
      "    </div></div>" +
      '  <div class="calc-license-period" data-sub-only id="term-field">' +
      '    <label for="term" data-i18n-calc="calcTerm">' + t("calcTerm") + '</label>' +
      '    <select id="term"></select></div>' +
      '  <div class="calc-license-period" data-perp-only id="perp-maint-field" style="display:none">' +
      '    <label for="perp_maint_years" data-i18n-calc="calcPerpMaintYears">' + t("calcPerpMaintYears") + "</label>" +
      '    <select id="perp_maint_years"></select></div>' +
      "</div>"
    );
  }

  function buildSummaryHtml(opts) {
    var t = opts.t;
    var money = opts.money;
    var model = opts.licenseModel || "subscription";
    var html = opts.linesHtml || "";

    if (opts.volByRequest) {
      html += '<div class="line"><span class="l disc">' + t("volByRequest") +
        '</span><span class="v disc">' + t("priceByRequest") + "</span></div>";
    } else if (opts.volDiscount) {
      html += '<div class="line"><span class="l disc">' + t("volDisc") +
        (opts.epCount != null ? " (" + opts.epCount + " " + t("volEndpoints") + ")" : "") +
        '</span><span class="v disc">−' + Math.round(opts.volDiscount * 100) + "%</span></div>";
    }

    if (model === "subscription") {
      var sub = opts.subscription;
      var termName = opts.termLabel(sub.term.key, sub.term.label);
      if (sub.term.discount) {
        html += '<div class="line"><span class="l disc">' + t("termDisc") + " (" + termName + ")" +
          '</span><span class="v disc">−' + Math.round(sub.term.discount * 100) + "%</span></div>";
      }
      html += '<div class="model-note">' + t("subModelNote") + "</div>";
      html += '<div class="total"><span class="l">' + t("totalYear") + '</span><span class="big">' + money(sub.totalYear) + "</span></div>";
      if (sub.years > 1) {
        html += '<div class="term-total"><span>' + t("totalTerm") + " " + sub.years + " " + t("years") +
          '</span><span><b>' + money(sub.totalTerm) + "</b></span></div>";
      }
    } else {
      var perp = opts.perpetual;
      html += '<div class="model-note">' + t("perpModelNote") + "</div>";
      html += '<div class="line"><span class="l">' + t("perpLicenseFee") +
        '</span><span class="v">' + money(perp.perpOnetime) + " " + t("perpOnce") + "</span></div>";
      html += '<div class="line"><span class="l">' + t("perpMaintLine") + " " + t("perYear") +
        '</span><span class="v">' + money(perp.perpMaintYear) + t("perYear") + "</span></div>";
      if (perp.maintYears > 1) {
        html += '<div class="term-total"><span>' + t("perpMaintFor") + " " + perp.maintYears + " " + t("years") +
          '</span><span><b>' + money(perp.perpMaintTotal) + "</b></span></div>";
      }
      html += '<div class="total"><span class="l">' + t("perpTotalYear1") +
        '</span><span class="big">' + money(perp.perpFirstYear) + "</span></div>";
      if (perp.maintYears > 1) {
        html += '<div class="term-total"><span>' + t("perpTotalAll") + " (" + t("perpOnce") + " + " + t("perpMaintFor") +
          " " + perp.maintYears + " " + t("years") + ')</span><span><b>' +
          money(perp.perpOnetime + perp.perpMaintTotal) + "</b></span></div>";
      }
    }

    return html;
  }

  function mailQuoteBody(opts) {
    var t = opts.t;
    var lines = [t("mailBody")];
    if (opts.ws != null) lines.push(t("mailWs") + ": " + opts.ws);
    if (opts.servers != null) lines.push(t("mailServers") + ": " + opts.servers);
    if (opts.users != null) lines.push(t("mailUsers") + ": " + opts.users);
    lines.push(t("mailLicenseModel") + ": " + (opts.licenseModel === "perpetual" ? t("licensePerpetual") : t("licenseSubscription")));
    if (opts.licenseModel === "perpetual") {
      lines.push(t("mailPerpOnetime") + ": " + opts.money(opts.perpetual.perpOnetime));
      lines.push(t("mailPerpMaint") + ": " + opts.money(opts.perpetual.perpMaintYear) + t("perYear"));
      if (opts.perpetual.maintYears > 1) {
        lines.push(t("perpMaintFor") + " " + opts.perpetual.maintYears + " " + t("years") + ": " +
          opts.money(opts.perpetual.perpMaintTotal));
      }
    } else {
      lines.push(t("mailTerm") + ": " + opts.termLabel);
      lines.push(t("mailApprox") + ": " + opts.money(opts.subscription.totalYear) + t("perYear"));
      if (opts.subscription.years > 1) {
        lines.push(t("totalTerm") + " " + opts.subscription.years + " " + t("years") + ": " +
          opts.money(opts.subscription.totalTerm));
      }
    }
    return lines.join("\n");
  }

  window.ERA_CALC_License = {
    perpCfg: perpCfg,
    termDiscounts: termDiscounts,
    termByKey: termByKey,
    yearsOf: yearsOf,
    subscriptionTotals: subscriptionTotals,
    perpetualTotals: perpetualTotals,
    syncLicenseModelUI: syncLicenseModelUI,
    bindLicenseModelUI: bindLicenseModelUI,
    fillMaintYearsSelect: fillMaintYearsSelect,
    licenseModelFieldsHtml: licenseModelFieldsHtml,
    buildSummaryHtml: buildSummaryHtml,
    mailQuoteBody: mailQuoteBody
  };
})();
