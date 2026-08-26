(function () {
  "use strict";

  // Syntax highlighting is done by the embedded PrismJS bundle (doc-templates/
  // prism.js), which registers the gad / gadt / gadx grammars and highlights every
  // `code[class*="language-…"]` on load. Nothing to do here for the code fences.
  function esc(s) {
    return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
  }

  // Build the sidebar table of contents from the rendered sections.
  var toc = document.getElementById("toc");
  if (toc) {
    var html = "";
    document.querySelectorAll(".doc-section").forEach(function (sec) {
      var h2 = sec.querySelector("h2");
      var title = h2 ? h2.textContent.trim() : "";
      html += "<div class='group' data-section='" + sec.id + "'>";
      html += "<div class='title'>" + esc(title) + "</div>";
      var syms = sec.querySelectorAll(".symbol");
      if (syms.length) {
        syms.forEach(function (sym) {
          var name = sym.getAttribute("data-name") || "";
          html += "<a href='#" + sym.id + "' data-target='" + sym.id + "'><code>" + esc(name) + "</code></a>";
        });
      } else if (sec.id !== "section-example") {
        html += "<div class='empty'>—</div>";
      }
      html += "</div>";
    });
    toc.innerHTML = html || "<div class='empty'>No symbols.</div>";
  }

  function setActive(id) {
    document.querySelectorAll("#toc a").forEach(function (a) {
      a.classList.toggle("active", a.getAttribute("data-target") === id);
    });
  }
  if (toc) {
    toc.addEventListener("click", function (e) {
      var a = e.target.closest("a"); if (!a) return;
      setActive(a.getAttribute("data-target"));
    });
  }

  var symbols = Array.prototype.slice.call(document.querySelectorAll(".symbol"));
  if (symbols.length && "IntersectionObserver" in window) {
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (en) { if (en.isIntersecting) setActive(en.target.id); });
    }, { rootMargin: "-60px 0px -70% 0px" });
    symbols.forEach(function (s) { io.observe(s); });
  }

  var search = document.getElementById("search");
  var content = document.querySelector(".content");
  if (search) {
    search.addEventListener("input", function () {
      var q = search.value.trim().toLowerCase();
      var anyVisible = false;
      document.querySelectorAll(".doc-section").forEach(function (sec) {
        var secHit = false;
        sec.querySelectorAll(".symbol").forEach(function (sym) {
          var hit = q === "" || sym.textContent.toLowerCase().indexOf(q) >= 0;
          sym.classList.toggle("hidden", !hit);
          if (hit) { secHit = true; anyVisible = true; }
        });
        if (!sec.querySelector(".symbol")) {
          secHit = q === "" || sec.textContent.toLowerCase().indexOf(q) >= 0;
          if (secHit) anyVisible = true;
        }
        sec.classList.toggle("hidden", !secHit);
      });
      document.querySelectorAll("#toc a").forEach(function (a) {
        var sym = document.getElementById(a.getAttribute("data-target"));
        a.style.display = (sym && sym.classList.contains("hidden")) ? "none" : "";
      });
      var nr = document.getElementById("no-results");
      if (!anyVisible && q !== "") {
        if (!nr) {
          nr = document.createElement("div");
          nr.id = "no-results"; nr.className = "no-results";
          content.appendChild(nr);
        }
        nr.textContent = 'No matches for "' + search.value + '".';
      } else if (nr) {
        nr.remove();
      }
    });
  }

  // Retractable index (sidebar): shown by default, toggled by the ☰ button and
  // remembered per viewer. `body.toc-hidden` hides it (see doc.css).
  var tocToggle = document.getElementById("toc-toggle");
  if (tocToggle) {
    var hidden = false;
    try { hidden = localStorage.getItem("gaddoc.toc") === "hidden"; } catch (e) {}
    function applyToc() {
      document.body.classList.toggle("toc-hidden", hidden);
      tocToggle.classList.toggle("active", !hidden);
      tocToggle.setAttribute("aria-expanded", hidden ? "false" : "true");
    }
    applyToc();
    tocToggle.addEventListener("click", function () {
      hidden = !hidden;
      try { localStorage.setItem("gaddoc.toc", hidden ? "hidden" : "shown"); } catch (e) {}
      applyToc();
    });
  }

  // Theme toggle. The IDE sets <html data-theme> to match its theme; the button
  // lets a standalone viewer flip it.
  var btn = document.getElementById("theme-toggle");
  if (btn) {
    btn.addEventListener("click", function () {
      var root = document.documentElement;
      var cur = root.getAttribute("data-theme");
      if (!cur) {
        cur = window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
      }
      root.setAttribute("data-theme", cur === "dark" ? "light" : "dark");
    });
  }
})();
