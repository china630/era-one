/* Mount full ERA Control pricing calculator (legacy portal logic). */
(function () {
  "use strict";

  var TEMPLATE =
    '<div class="calc">' +
    '  <div class="panel calc-form">' +
    '    <h3 data-i18n-calc="calcParams">Parameters</h3>' +
    '    <div class="calc-grid">' +
      '      <div class="field"><label for="ws" data-i18n-calc="calcWs">Workstations</label><input type="number" id="ws" min="0" /></div>' +
      '      <div class="field"><label for="servers" data-i18n-calc="calcServers">Servers (×3)</label><input type="number" id="servers" min="0" /></div>' +
      '      <div class="field field-full"><label for="bundle" data-i18n-calc="calcBundle">Bundle</label><select id="bundle"></select></div>' +
      '      <div class="field field-full" data-license-fields></div>' +
    '    </div>' +
    '    <h3 class="modules-title" data-i18n-calc="calcModules">Modules</h3>' +
    '    <div class="modules" id="modules"></div>' +
    '  </div>' +
    '  <div class="panel summary">' +
    '    <h3 data-i18n-calc="calcSummary">Estimated cost</h3>' +
    '    <div id="result"></div>' +
    '  </div>' +
    '</div>';

  window.ERA_mountControlCalc = function (container) {
    if (!container) return;
    container.innerHTML = '<div id="era-calc">' + TEMPLATE + "</div>";
    var licHost = container.querySelector("[data-license-fields]");
    var t = function (k) {
      return (window.ERA_LANG && window.ERA_LANG.t(k)) || k;
    };
    if (licHost && window.ERA_CALC_License) {
      licHost.innerHTML = window.ERA_CALC_License.licenseModelFieldsHtml(t);
      window.ERA_CALC_License.fillMaintYearsSelect(
        container.querySelector("#perp_maint_years"),
        { perpMaintYears: 1 },
        t
      );
    }
    if (window.ERA_CALC_applyStatic) window.ERA_CALC_applyStatic();
    if (window.ERA_CALC_init) window.ERA_CALC_init();
  };
})();
