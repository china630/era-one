/* ERA One — product catalog (SSOT for site navigation + datasheet mapping). */
window.ERA_CATALOG = (function () {
  var DS = "datasheets/";

  var PRODUCTS = {
    control: {
      slogan: "ONE AGENT. ONE PLATFORM. ONE CONTROL.",
      calc: "control",
      familyDs: "ERA-One-DataSheet.html",
      editions: [
        { n: "ERA Core", slug: "era-core", ds: "01-ERA-Core.html" },
        { n: "ERA Control AI", slug: "era-control-ai", ds: "02-ERA-Control-AI.html" },
        { n: "ERA Response", slug: "era-response", ds: "03-ERA-Response.html" },
        { n: "ERA Vuln", slug: "era-vuln", ds: "04-ERA-Vuln.html" },
        { n: "ERA Federated / National", slug: "era-federated-national", ds: "05-ERA-Federated-National.html" },
        { n: "ERA Workbench", slug: "era-workbench", ds: "06-ERA-Workbench.html" },
        { n: "ERA Exposure", slug: "era-exposure", ds: "07-ERA-Exposure.html" },
        { n: "ERA BYO-EDR Hub", slug: "era-byo-edr", ds: "08-ERA-BYO-EDR.html" },
        { n: "ERA Manage", slug: "era-manage", ds: "09-ERA-Manage.html" },
        { n: "ERA Service", slug: "era-service", ds: "10-ERA-Service.html" },
        { n: "ERA Provision", slug: "era-provision", ds: "11-ERA-Provision.html" },
        { n: "ERA PAM", slug: "era-pam", ds: "12-ERA-PAM.html" },
        { n: "ERA Observe", slug: "era-observe", ds: "13-ERA-Observe.html" },
        { n: "ERA Sovereign Hybrid", slug: "era-sovereign-hybrid", ds: "14-ERA-Sovereign-Hybrid.html" }
      ]
    },
    communications: {
      slogan: "ONE IDENTITY. ONE PLATFORM. ONE CONVERSATION.",
      calc: "users",
      familyDs: "ERA-Communications-DataSheet.html",
      editions: [
        { n: "ERA Mail Server", slug: "era-mail-server", ds: "comms-01-ERA-Mail-Server.html" },
        { n: "ERA Mail Client", slug: "era-mail-client", ds: "comms-02-ERA-Mail-Client.html" },
        { n: "ERA Conference", slug: "era-conference", ds: "comms-03-ERA-Conference.html" },
        { n: "ERA Chat", slug: "era-chat", ds: "comms-04-ERA-Chat.html" },
        { n: "ERA Comms AI", slug: "era-comms-ai", ds: "comms-05-ERA-Comms-AI.html" },
        { n: "ERA Mail Connect", slug: "era-mail-connect", ds: "comms-06-ERA-Mail-Connect.html" },
      ]
    },
    office: {
      slogan: "ONE WORKSPACE. ONE PLATFORM. ONE TEAM.",
      calc: "users",
      familyDs: "ERA-Office-DataSheet.html",
      editions: [
        { n: "ERA Drive", slug: "era-drive", ds: "office-00-ERA-Drive.html", tagKey: "ed.drive.tag" },
        { n: "ERA Documents", slug: "era-documents", ds: "office-01-ERA-Documents.html", tagKey: "ed.documents.tag" },
        { n: "ERA Tables", slug: "era-tables", ds: "office-02-ERA-Tables.html", tagKey: "ed.tables.tag" },
        { n: "ERA Presentations", slug: "era-presentations", ds: "office-03-ERA-Presentations.html", tagKey: "ed.presentations.tag" },
        { n: "ERA Projects", slug: "era-projects", ds: "office-04-ERA-Projects.html", tagKey: "ed.projects.tag" },
        { n: "ERA Office AI", slug: "era-office-ai", ds: "office-05-ERA-Office-AI.html", tagKey: "ed.officeAi.tag" }
      ]
    }
  };

  var FAMS = [
    { key: "control", name: "ERA Control", page: "control.html", tagKey: "hero.triControl" },
    { key: "communications", name: "ERA Communications", page: "communications.html", tagKey: "hero.triComms" },
    { key: "office", name: "ERA Office", page: "office.html", tagKey: "hero.triOffice" }
  ];

  function famOf(key) {
    for (var i = 0; i < FAMS.length; i++) if (FAMS[i].key === key) return FAMS[i];
    return null;
  }

  function moduleHref(familyKey, edition) {
    return "edition.html?id=" + encodeURIComponent(edition.slug);
  }

  function findEdition(slug) {
    for (var fk in PRODUCTS) {
      if (!PRODUCTS.hasOwnProperty(fk)) continue;
      for (var i = 0; i < PRODUCTS[fk].editions.length; i++) {
        var e = PRODUCTS[fk].editions[i];
        if (e.slug === slug) return { familyKey: fk, edition: e, family: famOf(fk) };
      }
    }
    return null;
  }

  return { DS: DS, PRODUCTS: PRODUCTS, FAMS: FAMS, famOf: famOf, moduleHref: moduleHref, findEdition: findEdition };
})();
