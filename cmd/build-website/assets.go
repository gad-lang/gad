package main

// layoutTemplate is the html/template for every page. Assets are referenced
// relatively so the site works at any base path (including /<commit-id>/).
const layoutTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Title}} · Gad</title>
<script>(function(){var s=localStorage.getItem("gad-theme");var t=s==="light"||s==="dark"?s:(window.matchMedia&&window.matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light");document.documentElement.dataset.theme=t;})();</script>
<link rel="stylesheet" href="styles.css">
</head>
<body>
<header class="site-header">
  <a class="brand" href="index.html">Gad</a>
  <div class="search"><input id="q" type="search" placeholder="Search docs…" autocomplete="off"><div id="results"></div></div>
  <button class="theme-toggle" id="theme">◐</button>
</header>
<div class="layout">
  <nav class="sidebar">
    {{range .Groups}}<div class="nav-group"><div class="nav-title">{{.Name}}</div>
      {{range .Pages}}<a class="nav-link{{if eq .OutFile $.Active}} active{{end}}" href="{{.OutFile}}">{{.Title}}</a>{{end}}
    </div>{{end}}
  </nav>
  <main class="content">{{.Content}}</main>
  {{if .TOC}}<aside class="toc"><div class="nav-title">On this page</div>{{range .TOC}}<a href="#{{.ID}}">{{.Text}}</a>{{end}}</aside>{{end}}
</div>
<footer class="site-footer">Gad — a fast, dynamic scripting language embedded in Go. Built with <code>cmd/build-website</code>.</footer>
<script src="theme.js"></script>
<script src="search.js"></script>
</body>
</html>`

// playgroundBody is the <main> content of the Playground page. play.js wires
// the WASM module + editor.
const playgroundBody = `<h1>Playground</h1>
<p>Edit Gad below and Format or Run it. Everything executes in your browser via WebAssembly.</p>
<div class="play">
  <div class="play-toolbar">
    <button id="run">Run ▶</button>
    <button id="fmt">Format</button>
    <span id="status" class="muted">loading WebAssembly…</span>
  </div>
  <textarea id="code" spellcheck="false">a := 1
b := 2
squares := [n*n for n in [1,2,3,4] if n > 1]
println("hello from gad", a + b, squares)
return squares
</textarea>
  <pre id="out" class="play-out"></pre>
</div>
<script src="wasm_exec.js"></script>
<script src="play.js"></script>`

const siteCSS = `:root{font-family:system-ui,sans-serif;color-scheme:light;
--bg:#fafafa;--panel:#fff;--border:#e2e2ea;--fg:#1d1d28;--muted:#6b6b80;--accent:#3b5bdb;--code-bg:#f3f3f7;}
:root[data-theme=dark]{color-scheme:dark;--bg:#1e1e2e;--panel:#252539;--border:#3a3a52;--fg:#e4e4f0;--muted:#9a9ab5;--accent:#8aa6ff;--code-bg:#2a2a3d;}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--fg);line-height:1.55}
a{color:var(--accent);text-decoration:none}a:hover{text-decoration:underline}
.site-header{display:flex;align-items:center;gap:1rem;padding:.6rem 1rem;border-bottom:1px solid var(--border);background:var(--panel);position:sticky;top:0;z-index:10}
.brand{font-weight:700;font-size:1.15rem}
.search{position:relative;flex:1;max-width:480px}
.search input{width:100%;padding:.4rem .6rem;border:1px solid var(--border);border-radius:6px;background:var(--bg);color:var(--fg)}
#results{position:absolute;top:110%;left:0;right:0;background:var(--panel);border:1px solid var(--border);border-radius:6px;display:none;max-height:60vh;overflow:auto;z-index:20}
#results a{display:block;padding:.5rem .7rem;border-bottom:1px solid var(--border)}
#results a:hover,#results a.sel{background:var(--code-bg);text-decoration:none}
#results .r-title{font-weight:600}#results .r-snip{color:var(--muted);font-size:.85rem}
.theme-toggle{background:transparent;border:1px solid var(--border);color:var(--fg);border-radius:6px;padding:.3rem .6rem;cursor:pointer}
.layout{display:grid;grid-template-columns:240px minmax(0,1fr) 200px;gap:1.5rem;max-width:1200px;margin:0 auto;padding:1.5rem 1rem}
.sidebar{position:sticky;top:64px;align-self:start;max-height:calc(100vh - 80px);overflow:auto}
.nav-group{margin-bottom:1rem}.nav-title{font-size:.72rem;text-transform:uppercase;letter-spacing:.05em;color:var(--muted);margin-bottom:.3rem}
.nav-link{display:block;padding:.18rem .4rem;border-radius:4px;color:var(--fg);font-size:.92rem}
.nav-link:hover{background:var(--code-bg);text-decoration:none}.nav-link.active{color:var(--accent);font-weight:600}
.content{min-width:0}.content h1{margin-top:0}
.content h1,.content h2,.content h3{line-height:1.25}.content h2{margin-top:1.8rem;border-bottom:1px solid var(--border);padding-bottom:.2rem}
.content code{background:var(--code-bg);padding:.1rem .3rem;border-radius:4px;font-size:.9em}
.content pre{background:var(--code-bg);padding:.8rem 1rem;border-radius:8px;overflow:auto;border:1px solid var(--border)}
.content pre code{background:none;padding:0}
.content table{border-collapse:collapse;width:100%;margin:1rem 0;display:block;overflow:auto}
.content th,.content td{border:1px solid var(--border);padding:.4rem .6rem;text-align:left}
.content th{background:var(--code-bg)}
.content blockquote{border-left:3px solid var(--accent);margin:1rem 0;padding:.2rem 1rem;color:var(--muted)}
.toc{position:sticky;top:64px;align-self:start;font-size:.85rem}
.toc a{display:block;padding:.12rem 0;color:var(--muted)}.toc a:hover{color:var(--accent)}
.site-footer{border-top:1px solid var(--border);padding:1rem;text-align:center;color:var(--muted);font-size:.85rem}
.muted{color:var(--muted)}
.play-toolbar{display:flex;gap:.5rem;align-items:center;margin:.6rem 0}
.play-toolbar button{background:var(--panel);border:1px solid var(--border);color:var(--fg);border-radius:6px;padding:.35rem .7rem;cursor:pointer}
#code{width:100%;height:280px;font-family:monospace;font-size:.9rem;padding:.7rem;border:1px solid var(--border);border-radius:8px;background:var(--code-bg);color:var(--fg)}
.play-out{min-height:80px;white-space:pre-wrap}
@media(max-width:900px){.layout{grid-template-columns:1fr}.sidebar,.toc{position:static;max-height:none}}`

const themeJS = `(function(){var b=document.getElementById("theme");if(!b)return;function cur(){return document.documentElement.dataset.theme==="dark"?"dark":"light"}b.textContent=cur()==="dark"?"☀":"☾";b.onclick=function(){var t=cur()==="dark"?"light":"dark";document.documentElement.dataset.theme=t;localStorage.setItem("gad-theme",t);b.textContent=t==="dark"?"☀":"☾"}})();`

const searchJS = `(function(){
var q=document.getElementById("q"),res=document.getElementById("results"),idx=null,sel=-1;
fetch("search.json").then(function(r){return r.json()}).then(function(d){idx=d});
function snippet(text,term){var i=text.toLowerCase().indexOf(term);if(i<0)return text.slice(0,120);var s=Math.max(0,i-40);return (s>0?"…":"")+text.slice(s,s+120)+"…"}
function render(items,term){if(!items.length){res.style.display="none";return}res.innerHTML=items.map(function(it){return '<a href="'+it.url+'"><div class="r-title">'+it.title+'</div><div class="r-snip">'+snippet(it.text,term).replace(/[<>]/g,"")+'</div></a>'}).join("");res.style.display="block";sel=-1}
q.addEventListener("input",function(){var term=q.value.trim().toLowerCase();if(!term||!idx){res.style.display="none";return}var items=idx.filter(function(it){return it.title.toLowerCase().indexOf(term)>=0||it.text.toLowerCase().indexOf(term)>=0}).slice(0,12);render(items,term)});
q.addEventListener("keydown",function(e){var links=res.querySelectorAll("a");if(!links.length)return;if(e.key==="ArrowDown"){sel=(sel+1)%links.length}else if(e.key==="ArrowUp"){sel=(sel-1+links.length)%links.length}else if(e.key==="Enter"){if(sel>=0){location.href=links[sel].href}return}else{return}e.preventDefault();links.forEach(function(l){l.classList.remove("sel")});links[sel].classList.add("sel")});
document.addEventListener("click",function(e){if(!res.contains(e.target)&&e.target!==q)res.style.display="none"});
})();`

const playJS = `(function(){
var out=document.getElementById("out"),status=document.getElementById("status");
function ready(){return typeof window.gadFormat==="function"}
var go=new Go();
WebAssembly.instantiateStreaming(fetch("gad.wasm"),go.importObject).then(function(r){go.run(r.instance);
  var t=setInterval(function(){if(ready()){clearInterval(t);status.textContent="ready";document.getElementById("run").disabled=false;document.getElementById("fmt").disabled=false}},30);
}).catch(function(e){status.textContent="failed to load WebAssembly: "+e});
function src(){return document.getElementById("code").value}
document.getElementById("run").onclick=function(){if(!ready())return;var r=JSON.parse(window.gadRun(src()));var s="";if(r.stdout)s+=r.stdout;if(r.stderr)s+=r.stderr;if(r.ok&&r.result)s+="⇦ "+r.result+"\n";if(r.diagnostics)r.diagnostics.forEach(function(d){s+=d.line+":"+d.column+" "+d.message+"\n"});out.textContent=s};
document.getElementById("fmt").onclick=function(){if(!ready())return;var r=JSON.parse(window.gadFormat(src()));if(r.ok){document.getElementById("code").value=r.source;out.textContent=""}else{out.textContent=(r.diagnostics||[]).map(function(d){return d.line+":"+d.column+" "+d.message}).join("\n")}};
document.getElementById("run").disabled=true;document.getElementById("fmt").disabled=true;
})();`
