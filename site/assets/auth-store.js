/* Prototype auth — corporate email registration (localStorage). */
window.ERA_AUTH = (function () {
  "use strict";
  var KEY = "era-user";
  var FREE_DOMAINS = [
    "gmail.com", "googlemail.com", "yahoo.com", "hotmail.com", "outlook.com",
    "live.com", "icloud.com", "mail.ru", "yandex.ru", "yandex.com", "bk.ru",
    "inbox.ru", "list.ru", "proton.me", "protonmail.com", "aol.com", "gmx.com"
  ];

  function isCorporateEmail(email) {
    if (!email || email.indexOf("@") < 1) return false;
    var domain = email.split("@").pop().toLowerCase().trim();
    return FREE_DOMAINS.indexOf(domain) === -1;
  }

  function getUser() {
    try {
      var raw = localStorage.getItem(KEY);
      return raw ? JSON.parse(raw) : null;
    } catch (e) {
      return null;
    }
  }

  function isLoggedIn() {
    return !!getUser();
  }

  function saveUser(data) {
    if (!isCorporateEmail(data.email)) return false;
    localStorage.setItem(KEY, JSON.stringify({
      name: data.name || "",
      org: data.org || "",
      email: data.email.trim().toLowerCase(),
      registeredAt: new Date().toISOString()
    }));
    return true;
  }

  function login(email) {
    var user = getUser();
    if (!user || user.email !== email.trim().toLowerCase()) return false;
    return true;
  }

  function logout() {
    localStorage.removeItem(KEY);
  }

  return {
    isCorporateEmail: isCorporateEmail,
    getUser: getUser,
    isLoggedIn: isLoggedIn,
    saveUser: saveUser,
    login: login,
    logout: logout,
    FREE_DOMAINS: FREE_DOMAINS
  };
})();
