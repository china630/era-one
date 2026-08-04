/* ERA Control shared chrome (P0 shell + Theme Matrix Phase B). */
(function (global) {
  var NAV = [
    { group: "SecOps", items: [
      { id: "soc", href: "/ui/control/", label: "SOC Home", mod: "core" },
      { id: "workbench", href: "/ui/control/workbench/", label: "Workbench", mod: "core" },
      { id: "ai", href: "/ui/control/ai/", label: "AI Investigate", mod: "ai" },
      { id: "response", href: "/ui/control/response/", label: "Response", mod: "response" },
      { id: "vuln", href: "/ui/control/vuln/", label: "Vuln", mod: "vm" },
    ]},
    { group: "IT-Ops", items: [
      { id: "manage", href: "/ui/control/manage/", label: "Manage", mod: "manage" },
      { id: "service", href: "/ui/control/service/", label: "Service", mod: "service" },
      { id: "provision", href: "/ui/control/provision/", label: "Provision", mod: "provision" },
      { id: "pam", href: "/ui/control/pam/", label: "PAM", mod: "pam" },
    ]},
    { group: "Edge", items: [
      { id: "observe", href: "/ui/control/observe/", label: "Observe", mod: "observe" },
      { id: "perimeter", href: "/ui/control/perimeter/", label: "Perimeter", mod: "perimeter" },
      { id: "resolve", href: "/ui/control/resolve/", label: "Resolve", mod: "resolve" },
      { id: "byo", href: "/ui/control/byo/", label: "BYO-EDR", mod: "core" },
    ]},
  ];

  function api(path, opts) {
    opts = opts || {};
    opts.headers = opts.headers || {};
    if (!opts.headers["Content-Type"] && opts.body) {
      opts.headers["Content-Type"] = "application/json";
    }
    return fetch(path, opts).then(function (r) {
      return r.text().then(function (t) {
        var j = null;
        try { j = t ? JSON.parse(t) : null; } catch (e) { j = { raw: t }; }
        return { ok: r.ok, status: r.status, json: j, text: t };
      });
    });
  }

  function x(svc, path, opts) {
    return api("/api/x/" + svc + path, opts);
  }

  function mountShell(activeId) {
    document.body.classList.add("era-control");
    document.body.setAttribute("data-line", "control");
    var side = document.createElement("aside");
    side.className = "era-sidebar";
    side.innerHTML = '<div class="era-brand" id="era-brand"></div><nav class="era-nav" id="era-nav"></nav>';
    document.body.insertBefore(side, document.body.firstChild);

    if (global.EraChrome && global.EraChrome.mountBrand) {
      global.EraChrome.mountBrand(document.getElementById("era-brand"), {
        line: "control",
        title: "ERA Control",
        subtitle: "lab console",
        href: "/ui/control/",
      });
    } else {
      var brand = document.getElementById("era-brand");
      brand.innerHTML = "ERA Control<span>lab console</span>";
    }

    var main = document.querySelector("main") || document.body;
    if (main.tagName !== "MAIN") {
      var m = document.createElement("main");
      m.className = "era-main";
      while (document.body.childNodes.length > 1) {
        var n = document.body.childNodes[1];
        if (n === side) continue;
        m.appendChild(n);
      }
      document.body.appendChild(m);
      main = m;
    } else {
      main.classList.add("era-main");
    }

    if (global.EraChrome && global.EraChrome.mountSwitcher) {
      global.EraChrome.mountSwitcher(side, {});
    }

    api("/api/v1/shell/config").then(function (res) {
      var mods = (res.json && res.json.modules) || {};
      var role = (res.json && res.json.role) || "unknown";
      var nav = document.getElementById("era-nav");
      NAV.forEach(function (g) {
        var label = document.createElement("div");
        label.className = "group";
        label.textContent = g.group;
        nav.appendChild(label);
        g.items.forEach(function (it) {
          if (mods[it.mod] === false) return;
          var a = document.createElement("a");
          a.href = it.href;
          a.textContent = it.label;
          a.setAttribute("data-mod", it.id);
          if (it.id === activeId) a.classList.add("active");
          nav.appendChild(a);
        });
      });
      var meta = document.getElementById("era-role-meta");
      if (meta) {
        if (global.EraChrome && global.EraChrome.mountAccount) {
          meta.id = "era-role-meta";
          meta.classList.add("era-user-chip");
          global.EraChrome.mountAccount(meta, {
            label: "role: " + role,
            role: role,
            showLamp: false,
            title: "Signed-in role",
          });
        } else {
          meta.textContent = "role: " + role;
        }
      }
    });
  }

  function setStatus(el, text, kind) {
    if (!el) return;
    el.textContent = text || "";
    el.className = "era-status" + (kind ? " " + kind : "");
  }

  global.EraControl = {
    mountShell: mountShell,
    api: api,
    x: x,
    setStatus: setStatus,
  };
})(window);
