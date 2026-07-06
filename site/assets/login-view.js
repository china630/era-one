/* Login — corporate email check (prototype). */
(function () {
  "use strict";
  var AUTH = window.ERA_AUTH;
  if (!AUTH) return;

  function nextUrl() {
    var p = new URLSearchParams(location.search).get("next");
    return p || "downloads.html";
  }

  document.addEventListener("DOMContentLoaded", function () {
    var form = document.querySelector(".login-card");
    if (!form) return;

    if (AUTH.isLoggedIn()) {
      location.href = nextUrl();
      return;
    }

    var note = document.createElement("p");
    note.style.textAlign = "center";
    note.style.marginTop = "14px";
    note.innerHTML = '<a href="register.html?next=' + encodeURIComponent(nextUrl()) + '">Register with corporate email</a>';
    form.appendChild(note);

    form.addEventListener("submit", function (e) {
      e.preventDefault();
      var email = (form.querySelector('[name="user"]') || {}).value || "";
      email = email.trim().toLowerCase();
      if (!AUTH.isCorporateEmail(email)) {
        alert("Use your corporate email. Register first if you do not have an account.");
        return;
      }
      if (AUTH.getUser() && AUTH.getUser().email === email) {
        location.href = nextUrl();
        return;
      }
      if (!AUTH.getUser()) {
        location.href = "register.html?next=" + encodeURIComponent(nextUrl());
        return;
      }
      alert("Email not found. Please register first.");
    });
  });
})();
