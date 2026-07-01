(function(){
var q=document.getElementById("q"),res=document.getElementById("results"),idx=null,sel=-1;
fetch("search.json").then(function(r){return r.json()}).then(function(d){idx=d});
function snippet(text,term){var i=text.toLowerCase().indexOf(term);if(i<0)return text.slice(0,120);var s=Math.max(0,i-40);return (s>0?"…":"")+text.slice(s,s+120)+"…"}
function render(items,term){if(!items.length){res.style.display="none";return}res.innerHTML=items.map(function(it){return '<a href="'+it.url+'"><div class="r-title">'+it.title+'</div><div class="r-snip">'+snippet(it.text,term).replace(/[<>]/g,"")+'</div></a>'}).join("");res.style.display="block";sel=-1}
q.addEventListener("input",function(){var term=q.value.trim().toLowerCase();if(!term||!idx){res.style.display="none";return}var items=idx.filter(function(it){return it.title.toLowerCase().indexOf(term)>=0||it.text.toLowerCase().indexOf(term)>=0}).slice(0,12);render(items,term)});
q.addEventListener("keydown",function(e){var links=res.querySelectorAll("a");if(!links.length)return;if(e.key==="ArrowDown"){sel=(sel+1)%links.length}else if(e.key==="ArrowUp"){sel=(sel-1+links.length)%links.length}else if(e.key==="Enter"){if(sel>=0){location.href=links[sel].href}return}else{return}e.preventDefault();links.forEach(function(l){l.classList.remove("sel")});links[sel].classList.add("sel")});
document.addEventListener("click",function(e){if(!res.contains(e.target)&&e.target!==q)res.style.display="none"});
})();