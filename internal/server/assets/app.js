// Client-side progress (localStorage only — no server tracking), code-copy
// buttons, and Mermaid diagram rendering.
(function () {
  "use strict";

  var cfg = window.WORKSHOP || { id: "workshop", checkpoints: [], stepPath: "" };
  var KEY = "workshopify:" + cfg.id + ":done";

  function loadDone() {
    try {
      return new Set(JSON.parse(localStorage.getItem(KEY)) || []);
    } catch (e) {
      return new Set();
    }
  }
  function saveDone(set) {
    localStorage.setItem(KEY, JSON.stringify(Array.from(set)));
  }

  // Auto-mark the current checkpoint complete on visit.
  if (cfg.stepPath) {
    var done = loadDone();
    if (!done.has(cfg.stepPath)) {
      done.add(cfg.stepPath);
      saveDone(done);
    }
  }

  // Reactive progress component (header bar + index ticks).
  document.addEventListener("alpine:init", function () {
    window.Alpine.data("handbook", function () {
      return {
        done: 0,
        total: (cfg.checkpoints || []).length,
        pct: 0,
        init: function () {
          this.recompute();
          var self = this;
          window.addEventListener("storage", function () {
            self.recompute();
          });
        },
        recompute: function () {
          var set = loadDone();
          var cps = cfg.checkpoints || [];
          this.total = cps.length;
          this.done = cps.filter(function (p) {
            return set.has(p);
          }).length;
          this.pct = this.total ? Math.round((this.done / this.total) * 100) : 0;
        },
        isDone: function (path) {
          return loadDone().has(path);
        },
        reset: function () {
          localStorage.removeItem(KEY);
          location.reload();
        },
      };
    });
  });

  // Progressive enhancements that don't need Alpine.
  document.addEventListener("DOMContentLoaded", function () {
    document.querySelectorAll(".code-block").forEach(function (block) {
      var pre = block.querySelector("pre");
      if (!pre) return;
      var btn = document.createElement("button");
      btn.type = "button";
      btn.className = "copy-btn";
      btn.textContent = "Copy";
      btn.addEventListener("click", function () {
        navigator.clipboard.writeText(pre.innerText).then(function () {
          btn.textContent = "Copied!";
          setTimeout(function () {
            btn.textContent = "Copy";
          }, 1200);
        });
      });
      block.appendChild(btn);
    });

    if (window.mermaid) {
      window.mermaid.initialize({ startOnLoad: false, theme: "neutral" });
      window.mermaid.run({ querySelector: "pre.mermaid" });
    }
  });
})();
