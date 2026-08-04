/* Downloads page — trial packages per product family & edition. */
(function () {
  "use strict";
  var CAT = window.ERA_CATALOG;
  var AUTH = window.ERA_AUTH;

  function t(k) {
    var I = window.ERA_I18N || {};
    var lang = document.documentElement.lang || "en";
    var dict = I[lang] || I.en || {};
    return dict[k] != null ? dict[k] : (I.en && I.en[k] != null ? I.en[k] : k);
  }

  function trialMailSubject(name) {
    return "ERA One trial: " + name;
  }

  function handleTrial(name, pkg) {
    if (!AUTH || !AUTH.isLoggedIn()) {
      location.href = "register.html?next=" + encodeURIComponent("downloads.html");
      return;
    }
    var user = AUTH.getUser();
    var body = "Product: " + name + "\nPackage: " + pkg + "\nAccount: " + user.email;
    location.href = "mailto:sales@era-one.solutions?subject=" +
      encodeURIComponent(trialMailSubject(name)) + "&body=" + encodeURIComponent(body);
  }

  function renderBanner(container) {
    if (!container || !AUTH) return;
    if (AUTH.isLoggedIn()) {
      var u = AUTH.getUser();
      container.innerHTML =
        '<div class="dl-banner dl-banner-ok">' +
        '<span>' + t("dl.loggedAs") + ' <b>' + u.email + '</b></span>' +
        '<button type="button" class="btn-sm" id="dl-logout">' + t("dl.logout") + '</button>' +
        '</div>';
      var btn = document.getElementById("dl-logout");
      if (btn) btn.addEventListener("click", function () {
        AUTH.logout();
        location.reload();
      });
      return;
    }
    container.innerHTML =
      '<div class="dl-banner">' +
      '<p>' + t("dl.gateLead") + '</p>' +
      '<a class="btn-sm btn-primary" href="register.html?next=downloads.html">' + t("dl.register") + '</a>' +
      ' <a class="btn-sm" href="login.html?next=downloads.html">' + t("nav.login") + '</a>' +
      '</div>';
  }

  function renderList(container) {
    if (!container || !CAT) return;
    var html = "";
    CAT.FAMS.forEach(function (fam) {
      var p = CAT.PRODUCTS[fam.key];
      if (!p) return;
      html += '<div class="dl-family">';
      html += '<div class="dl-family-head"><h2>' + fam.name + '</h2>';
      html += '<span class="dl-trial-badge">' + t("dl.trialBadge") + '</span></div>';
      html += '<p class="dl-family-desc">' + t("dl.family." + fam.key) + '</p>';
      html += '<div class="dl-actions">';
      html += '<button type="button" class="pdf-btn dl-trial-btn" data-name="' + fam.name + '" data-pkg="' + fam.key + '-trial-bundle">' +
        t("dl.trialFamily") + '</button>';
      html += '</div>';
      html += '<div class="dl-editions">';
      p.editions.forEach(function (e) {
        html += '<div class="dl-ed-row">';
        html += '<span class="dl-ed-name">' + e.n + '</span>';
        html += '<button type="button" class="btn-sm dl-trial-btn" data-name="' + e.n + '" data-pkg="' + e.slug + '-trial">' +
          t("dl.trialModule") + '</button>';
        html += '</div>';
      });
      html += '</div></div>';
    });
    container.innerHTML = html;
    container.querySelectorAll(".dl-trial-btn").forEach(function (btn) {
      btn.addEventListener("click", function () {
        handleTrial(btn.getAttribute("data-name"), btn.getAttribute("data-pkg"));
      });
    });
  }

  document.addEventListener("DOMContentLoaded", function () {
    renderBanner(document.getElementById("downloads-banner"));
    renderList(document.getElementById("downloads-list"));
  });

  window.addEventListener("era-lang-changed", function () {
    renderBanner(document.getElementById("downloads-banner"));
    renderList(document.getElementById("downloads-list"));
  });
})();
