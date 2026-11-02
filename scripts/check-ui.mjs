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
for(const feature of ['function officialCostumes(', 'function costumeCard(', 'function tagModOutfit(', 'function ownCostume(', "api('official-outfit/random'", "data-mod-outfit", "data-own-costume", "pick-outfit-photo"]){
 assert.ok(html.includes(feature),"Missing official outfit UI feature: "+feature);
}
assert.ok(!html.includes('Aparência original'),"Generic original appearance card must not be shown");

const cardStart=html.indexOf("function costumeCard(agent,costume){");
const cardEnd=html.indexOf("function agentCard(name){",cardStart);
assert.ok(cardStart>=0&&cardEnd>cardStart,"Costume card source not found");
const cardSource=html.slice(cardStart,cardEnd);
const previewMods=Array.from({length:11},(_,i)=>({
  id:"mod-"+i,name:"Remielle Skin "+i,enabled:i===10,
  preview:true,outfitId:"base"
}));
const cardContext={
  state:{selectedOutfits:{},ownedOutfits:{}},
  modsForAgent:()=>previewMods,
  officialArt:()=>'<img src="/portrait.png">',
  outfitPhotoKey:(a,b)=>a+":"+b,
  customImageNames:new Set(),
  esc:v=>String(v??"").replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;"),
  portraitURL:()=>"/portrait.png",
  portraitData:()=>"/fallback.png",
  previewURL:()=>"/preview.png",
  expandedOfficialModLists:new Set(),
  officialModSearchQueries:new Map(),
  officialModPreviewLimit:3,
  outfitListKey:(a,b)=>a+"\\u001f"+b,
  normalizeModSearch:v=>String(v||"").normalize("NFD").replace(/[\\u0300-\\u036f]/g,"").toLowerCase()
};
const ctx=vm.createContext(cardContext);
vm.runInContext(cardSource+"\\nfunction renderFixture(){return costumeCard('Remielle',{id:'base',base:true,name:'Traje padrão'});}",ctx);
let htmlCollapsed=vm.runInContext("renderFixture()",ctx);
const countMods=src=>(src.match(/<div class="official-mod" role="listitem"/g)||[]).length;
assert.equal(countMods(htmlCollapsed),3,"More than 3 mods rendered in collapsed outfit");
assert.ok(htmlCollapsed.includes("Remielle Skin 10"),"Active mod must appear in top three");
assert.ok(htmlCollapsed.includes("Ver todos os 11 mods"),"Expand action missing");
assert.ok(!htmlCollapsed.includes("data-outfit-mod-search"),"Search should be hidden in collapsed state");
vm.runInContext("expandedOfficialModLists.add(outfitListKey('Remielle','base'))",ctx);
let htmlExpanded=vm.runInContext("renderFixture()",ctx);
assert.equal(countMods(htmlExpanded),11,"Expanded outfit must show all mods in scrollable list");
assert.ok(htmlExpanded.includes('aria-expanded="true"')&&htmlExpanded.includes("data-outfit-mod-search"),"Expanded controls missing");
vm.runInContext("officialModSearchQueries.set(outfitListKey('Remielle','base'),'skin 7')",ctx);
let htmlSearched=vm.runInContext("renderFixture()",ctx);
assert.equal(countMods(htmlSearched),1,"Expanded search must filter mods without changing selected outfit");
assert.ok(htmlSearched.includes("Remielle Skin 7")&&htmlSearched.includes("1 de 11"),"Search result count wrong");
cardContext.modsForAgent=()=>[];
let htmlEmpty=vm.runInContext("renderFixture()",ctx);
assert.ok(htmlEmpty.includes("Nenhum mod vinculado")&&!htmlEmpty.includes('data-action="toggle-outfit-mods"'),"Empty outfits must not show list expansion");
assert.ok(html.includes(".official-mod-list.is-expanded{max-height:300px")&&html.includes("if(a==='toggle-outfit-mods')")&&html.includes("data-outfit-mod-search"),"Bounded list or event handling missing");

console.log("UI JavaScript, official outfit gallery, and mod grouping syntax OK");
