(function(){
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
})();