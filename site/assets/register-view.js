/* Corporate registration form (prototype). */
(function () {
  "use strict";
  var AUTH = window.ERA_AUTH;
  if (!AUTH) return;

  function nextUrl() {
    var p = new URLSearchParams(location.search).get("next");
    return p || "downloads.html";
  }

  document.addEventListener("DOMContentLoaded", function () {
    var form = document.getElementById("register-form");
    var err = document.getElementById("reg-error");
    if (!form) return;

    form.addEventListener("submit", function (e) {
      e.preventDefault();
      var fd = new FormData(form);
      var email = (fd.get("email") || "").toString().trim();
      if (!AUTH.isCorporateEmail(email)) {
        if (err) err.hidden = false;
        return;
      }
      if (err) err.hidden = true;
      AUTH.saveUser({
        name: (fd.get("name") || "").toString().trim(),
        org: (fd.get("org") || "").toString().trim(),
        email: email
      });
      location.href = nextUrl();
    });
  });
})();
