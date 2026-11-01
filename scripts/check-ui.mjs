import fs from "node:fs";
import vm from "node:vm";
import assert from "node:assert/strict";

const html = fs.readFileSync("ui.html", "utf8");
const scripts = [...html.matchAll(/<script(?:\s[^>]*)?>([\s\S]*?)<\/script>/gi)].map(m => m[1]).filter(Boolean);
assert.ok(scripts.length, "Inline script not found");
for (const script of scripts) new vm.Script(script, { filename: "ui.html" });
for (const value of ["id=\"agentPicker\"", "id=\"agentPickerSearch\"", "data-action=\"select-agent\"", "function modHotkeysPanel(", "function renderPickerResults("]) {
  assert.ok(html.includes(value), "Missing UI feature: " + value);
}
console.log("UI JavaScript syntax and feature markers OK");
