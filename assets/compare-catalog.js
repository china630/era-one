/* ERA One — head-to-head compare catalog (by product family) */
window.ERA_COMPARE = (function () {
  var DS = "compare/";
  var FAMS = [
    { key: "control", name: "ERA Control", labelKey: "compare.famControl" },
    { key: "communications", name: "ERA Communications", labelKey: "compare.famComms" },
    { key: "office", name: "ERA Office", labelKey: "compare.famOffice" }
  ];
  var items = [
    { id: "manageengine", name: "ManageEngine", family: "control", ds: "ERA-vs-ManageEngine.html" },
    { id: "ivanti", name: "Ivanti", family: "control", ds: "ERA-vs-Ivanti.html" },
    { id: "trendmicro", name: "Trend Micro", family: "control", ds: "ERA-vs-TrendMicro.html" },
    { id: "bigfix", name: "HCL BigFix", family: "control", ds: "ERA-vs-BigFix.html" },
    { id: "positivetech", name: "Positive Technologies", family: "control", ds: "ERA-vs-PositiveTechnologies.html" },
    { id: "checkpoint", name: "Check Point", family: "control", ds: "ERA-vs-CheckPoint.html" },
    { id: "exchange", name: "Microsoft Exchange", family: "communications", ds: "ERA-vs-Exchange.html" },
    { id: "microsoft365", name: "Microsoft 365", family: "communications", ds: "ERA-vs-Microsoft365-Comms.html" },
    { id: "icewarp", name: "IceWarp", family: "communications", ds: "ERA-vs-IceWarp.html" },
    { id: "communigate", name: "CommuniGate Pro", family: "communications", ds: "ERA-vs-CommuniGate.html" },
    { id: "microsoft365office", name: "Microsoft 365", family: "office", ds: "ERA-vs-Microsoft365-Office.html" },
    { id: "googleworkspace", name: "Google Workspace", family: "office", ds: "ERA-vs-GoogleWorkspace.html" },
    { id: "onlyoffice", name: "OnlyOffice", family: "office", ds: "ERA-vs-OnlyOffice.html" }
  ];
  function find(id) {
    for (var i = 0; i < items.length; i++) if (items[i].id === id) return items[i];
    return null;
  }
  function byFamily(family) {
    return items.filter(function (it) { return it.family === family; });
  }
  function familyName(key) {
    for (var i = 0; i < FAMS.length; i++) if (FAMS[i].key === key) return FAMS[i].name;
    return key;
  }
  return { DS: DS, FAMS: FAMS, items: items, find: find, byFamily: byFamily, familyName: familyName };
})();
