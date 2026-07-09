// Юнит-тесты калькулятора портала (ADR-0021). Запуск: node site/test/calculator.test.js
"use strict";
const assert = require("assert");
const path = require("path");

global.window = global;
require(path.join(__dirname, "..", "pricing-data.js"));
require(path.join(__dirname, "..", "assets", "calc-license-model.js"));
const calc = require(path.join(__dirname, "..", "assets", "app.js"));

function approx(a, b, eps) { return Math.abs(a - b) <= (eps || 0.5); }
let failed = 0;
function check(name, cond) {
  if (cond) { console.log("  PASS " + name); }
  else { console.error("  FAIL " + name); failed++; }
}

// --- Сценарий 1: пакет SecOps, 1000 ПК + 50 серверов, 3 года предоплата ---
calc.setState({
  ws: 1000, servers: 50, term: "3y_prepaid",
  licenseModel: "subscription", perpMaintYears: 1,
  bundle: "secops", selected: { core: true, "control-ai": true, response: true }, qty: {}
});
let r = calc.compute();
check("S1 lines = 3", r.lines.length === 3);
check("S1 subtotal ~ 17664", approx(r.subtotal, 17664));
check("S1 totalYear ~ 14131.2", approx(r.totalYear, 14131.2));
check("S1 totalTerm ~ 42393.6 (3 года)", approx(r.totalTerm, 42393.6));
check("S1 volume tier 20%", r.vol.discount === 0.20);
check("S1 perpetual onetime = 3x subtotal", approx(r.perpetual.perpOnetime, 17664 * 3));
check("S1 perpetual maint/year = 20% subtotal", approx(r.perpetual.perpMaintYear, 17664 * 0.2));

// --- Сценарий 1b: perpetual, maintenance 3 года ---
calc.setState({ licenseModel: "perpetual", perpMaintYears: 3 });
r = calc.compute();
check("S1b perp maint total 3y", approx(r.perpetual.perpMaintTotal, 17664 * 0.2 * 3));
check("S1b perp year1 = 3x + 20%", approx(r.perpetual.perpFirstYear, 17664 * 3.2));

// --- Сценарий 2: только Core, 100 ПК, 1 год ---
calc.setState({
  ws: 100, servers: 0, term: "1y", licenseModel: "subscription",
  bundle: "", selected: {}, qty: {}
});
r = calc.compute();
check("S2 core only", r.lines.length === 1 && r.lines[0].key === "core");
check("S2 totalYear = 1200", approx(r.totalYear, 1200));
check("S2 perpetual onetime = 3600", approx(r.perpetual.perpOnetime, 3600));

// --- Сценарий 3: AI flat ---
calc.setState({
  ws: 5000, servers: 0, term: "1y", licenseModel: "subscription",
  bundle: "", selected: { "control-ai": true }, qty: {}
});
r = calc.compute();
let aiLine = r.lines.find(l => l.key === "control-ai");
check("S3 Control AI per-endpoint < flat, с объёмом = 32000", approx(aiLine.reg, 32000));

calc.setState({
  ws: 1000, servers: 0, term: "1y", licenseModel: "subscription",
  bundle: "", selected: { "control-ai": true }, qty: {}
});
r = calc.compute();
aiLine = r.lines.find(l => l.key === "control-ai");
check("S4 Control AI per-endpoint < flat, с объёмом = 7200", approx(aiLine.reg, 7200));

// --- Сценарий 5: PAM ---
calc.setState({
  ws: 200, servers: 0, term: "1y", licenseModel: "subscription",
  bundle: "", selected: { pam: true }, qty: { pam: 5, pam_target: 10 }
});
r = calc.compute();
let pam = r.lines.find(l => l.key === "pam");
let tgt = r.lines.find(l => l.key === "pam_target");
check("S5 PAM admins = 250", pam && approx(pam.reg, 250));
check("S5 PAM targets = 300", tgt && approx(tgt.reg, 300));

// --- Сценарий 6: объём 25000+ ---
calc.setState({
  ws: 30000, servers: 0, term: "1y", licenseModel: "subscription",
  bundle: "", selected: {}, qty: {}
});
r = calc.compute();
check("S6 volume byRequest at 30000", r.vol.byRequest === true);

// --- ERA_CALC_License helpers ---
const lic = global.ERA_CALC_License;
check("S9 subscription 1y no discount", approx(lic.subscriptionTotals(1000, "1y").totalYear, 1000));
check("S9 subscription 3y prepaid -20%", approx(lic.subscriptionTotals(1000, "3y_prepaid").totalYear, 800));
check("S9 perpetual 3x + maint", approx(lic.perpetualTotals(1000, 1).perpFirstYear, 3200));

// --- Product-line calculators ---
require(path.join(__dirname, "..", "assets", "calc-product.js"));
const comms = window.ERA_computeProductLine("communications", {
  users: 100, bundle: "comms-full", manual: false, selected: {},
  licenseModel: "subscription", term: "1y", perpMaintYears: 1
});
check("S7 comms-full 100 users ~ 2370", approx(comms.subtotal, 2370));
check("S7 comms subscription year = subtotal", approx(comms.subscription.totalYear, 2370));
check("S7 comms perpetual onetime = 3x", approx(comms.perpetual.perpOnetime, 2370 * 3));

const office = window.ERA_computeProductLine("office", {
  users: 500, bundle: "office-suite", manual: false, selected: {},
  licenseModel: "subscription", term: "3y_prepaid", perpMaintYears: 1
});
check("S8 office-suite 500 users ~ 8625", approx(office.subtotal, 8625));
check("S8 office 3y prepaid -20%", approx(office.subscription.totalYear, 8625 * 0.8));

if (failed) { console.error("\n" + failed + " тест(ов) упало"); process.exit(1); }
console.log("\nВсе тесты калькулятора пройдены.");
