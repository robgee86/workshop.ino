(function () {
  "use strict";

  var cfg = window.WORKSHOP || { id: "workshop", checkpoints: [], stepPath: "" };
  var KEY = "workshop.ino:" + cfg.id + ":done";

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

    function postAction(btn, url, opts) {
      if (opts.confirm && !window.confirm(opts.confirm)) return;
      var payload = {
        step: btn.getAttribute("data-step"),
        index: parseInt(btn.getAttribute("data-index"), 10),
      };
      var original = btn.textContent;
      btn.disabled = true;
      btn.textContent = opts.busy;
      fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })
        .then(function (r) {
          return r.json().then(function (j) {
            return { ok: r.ok && j.ok, error: j.error };
          });
        })
        .then(function (res) {
          if (res.ok) {
            btn.textContent = opts.done;
            btn.classList.add("action-ok");
          } else {
            btn.textContent = "Failed";
            btn.classList.add("action-err");
            window.alert(opts.label + " failed:\n\n" + (res.error || "unknown error"));
          }
        })
        .catch(function (e) {
          btn.textContent = "Failed";
          btn.classList.add("action-err");
          window.alert(opts.label + " failed:\n\n" + e);
        })
        .then(function () {
          setTimeout(function () {
            btn.disabled = false;
            btn.textContent = original;
            btn.classList.remove("action-ok", "action-err");
          }, 2500);
        });
    }

    document.querySelectorAll(".solution-btn").forEach(function (btn) {
      btn.addEventListener("click", function () {
        postAction(btn, "/apply-solution", {
          busy: "Applying…",
          done: "Applied!",
          label: "Apply solution",
          confirm:
            "Apply the solution? It replaces the current files in the app folder.",
        });
      });
    });

    // Clicking a diff file name downloads the .patch; don't also toggle the card.
    document.querySelectorAll(".diff-path").forEach(function (a) {
      a.addEventListener("click", function (e) {
        e.stopPropagation();
      });
    });

    if (window.mermaid) {
      window.mermaid.initialize({ startOnLoad: false, theme: "neutral" });
      window.mermaid.run({ querySelector: "pre.mermaid" });
    }

    // Keyboard navigation between steps. Left = previous, Right = next.
    var prevLink = document.querySelector('a[rel="prev"]');
    var nextLink = document.querySelector('a[rel="next"]');
    if (prevLink || nextLink) {
      document.addEventListener("keydown", function (e) {
        if (e.ctrlKey || e.metaKey || e.altKey || e.shiftKey) return;
        var el = e.target;
        if (el && el.closest && el.closest('input, textarea, select, [contenteditable="true"]')) {
          return;
        }
        if (e.key === "ArrowRight" && nextLink) {
          e.preventDefault();
          window.location.href = nextLink.href;
        } else if (e.key === "ArrowLeft" && prevLink) {
          e.preventDefault();
          window.location.href = prevLink.href;
        }
      });
    }
  });
})();
