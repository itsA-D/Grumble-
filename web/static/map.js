// World map interaction (FR-MAP-1): pan, zoom and region presets. No inline
// script so the CSP can stay strict. Everything reads from the rendered SVG;
// the server owns the data.
(function () {
  "use strict";

  var WORLD = { x: 0, y: 0, w: 880, h: 440 };
  var MIN_ZOOM = 1;
  var MAX_ZOOM = 8;

  function clamp(v, lo, hi) {
    return Math.max(lo, Math.min(hi, v));
  }

  function viewFor(zoom, cx, cy) {
    var w = WORLD.w / zoom;
    var h = WORLD.h / zoom;
    return clampView({ x: cx - w / 2, y: cy - h / 2, w: w, h: h });
  }

  function clampView(v) {
    v.w = clamp(v.w, WORLD.w / MAX_ZOOM, WORLD.w);
    v.h = clamp(v.h, WORLD.h / MAX_ZOOM, WORLD.h);
    v.x = clamp(v.x, 0, WORLD.w - v.w);
    v.y = clamp(v.y, 0, WORLD.h - v.h);
    return v;
  }

  function zoomOf(v) {
    return WORLD.w / v.w;
  }

  function initMap() {
    var svg = document.getElementById("map-svg");
    if (!svg || svg.dataset.bound === "1") {
      return;
    }
    svg.dataset.bound = "1";

    var view = { x: 0, y: 0, w: 880, h: 440 };
    var hint = document.getElementById("map-hint");
    var tooltip = document.getElementById("map-tooltip");
    var frame = svg.parentElement;
    var reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

    function apply() {
      view = clampView(view);
      svg.setAttribute("viewBox", view.x + " " + view.y + " " + view.w + " " + view.h);
      if (hint) {
        hint.textContent = zoomOf(view).toFixed(1) + "x / drag to pan / scroll to zoom";
      }
    }

    function applyViewBox(box) {
      view = clampView({ x: box[0], y: box[1], w: box[2], h: box[3] });
      apply();
    }

    function animateTo(box) {
      if (reduced) {
        applyViewBox(box);
        return;
      }
      var start = { x: view.x, y: view.y, w: view.w, h: view.h };
      var target = { x: box[0], y: box[1], w: box[2], h: box[3] };
      var t0 = performance.now();
      function step(t) {
        var k = clamp((t - t0) / 250, 0, 1);
        var e = 1 - Math.pow(1 - k, 3);
        view = {
          x: start.x + (target.x - start.x) * e,
          y: start.y + (target.y - start.y) * e,
          w: start.w + (target.w - start.w) * e,
          h: start.h + (target.h - start.h) * e,
        };
        apply();
        if (k < 1) {
          requestAnimationFrame(step);
        }
      }
      requestAnimationFrame(step);
    }

    // Convert a pointer position to world coordinates in the current view.
    function toWorld(clientX, clientY) {
      var r = svg.getBoundingClientRect();
      // Preserve the SVG's meet scaling inside its box.
      var scale = Math.min(r.width / view.w, r.height / view.h);
      var offX = (r.width - view.w * scale) / 2;
      var offY = (r.height - view.h * scale) / 2;
      return {
        x: view.x + (clientX - r.left - offX) / scale,
        y: view.y + (clientY - r.top - offY) / scale,
      };
    }

    function zoomAt(factor, clientX, clientY) {
      var before = zoomOf(view);
      var after = clamp(before * factor, MIN_ZOOM, MAX_ZOOM);
      if (after === before) {
        return;
      }
      var r = svg.getBoundingClientRect();
      var cx = clientX == null ? r.left + r.width / 2 : clientX;
      var cy = clientY == null ? r.top + r.height / 2 : clientY;
      var p = toWorld(cx, cy);
      var ratio = after / before;
      var nw = view.w / ratio;
      var nh = view.h / ratio;
      var rx = (p.x - view.x) / view.w;
      var ry = (p.y - view.y) / view.h;
      view = clampView({ x: p.x - rx * nw, y: p.y - ry * nh, w: nw, h: nh });
      apply();
    }

    svg.addEventListener(
      "wheel",
      function (e) {
        e.preventDefault();
        zoomAt(Math.exp(-e.deltaY * 0.0015), e.clientX, e.clientY);
      },
      { passive: false }
    );

    var dragging = false;
    var moved = false;
    var last = null;

    svg.addEventListener("pointerdown", function (e) {
      if (e.button !== 0) {
        return;
      }
      dragging = true;
      moved = false;
      last = { x: e.clientX, y: e.clientY };
      // Capture only once a drag actually starts: capturing on pointerdown
      // retargets the compatibility click to the SVG, so a tap on a country
      // would never reach its link.
    });

    svg.addEventListener("pointermove", function (e) {
      if (!dragging) {
        return;
      }
      var dx = e.clientX - last.x;
      var dy = e.clientY - last.y;
      if (!moved && Math.abs(dx) + Math.abs(dy) > 2) {
        moved = true;
        svg.setPointerCapture(e.pointerId);
        svg.classList.add("is-panning");
      }
      if (!moved) {
        return;
      }
      var r = svg.getBoundingClientRect();
      var scale = Math.min(r.width / view.w, r.height / view.h) || 1;
      view = clampView({ x: view.x - dx / scale, y: view.y - dy / scale, w: view.w, h: view.h });
      last = { x: e.clientX, y: e.clientY };
      apply();
    });

    function endDrag(e) {
      if (!dragging) {
        return;
      }
      dragging = false;
      svg.classList.remove("is-panning");
      if (svg.hasPointerCapture && svg.hasPointerCapture(e.pointerId)) {
        svg.releasePointerCapture(e.pointerId);
      }
    }
    svg.addEventListener("pointerup", endDrag);
    svg.addEventListener("pointercancel", endDrag);

    // A drag must not also follow the link. Suppress the click once when the
    // pointer moved more than the small threshold.
    svg.addEventListener(
      "click",
      function (e) {
        if (moved) {
          e.preventDefault();
          e.stopPropagation();
          moved = false;
        }
      },
      true
    );

    document.getElementById("map-zoom-in").addEventListener("click", function () {
      zoomAt(1.5);
    });
    document.getElementById("map-zoom-out").addEventListener("click", function () {
      zoomAt(1 / 1.5);
    });
    document.getElementById("map-reset").addEventListener("click", function () {
      animateTo([0, 0, 880, 440]);
      setActiveRegion("world");
    });

    // Tooltip: show the country name and its pain count on hover/focus.
    function showTip(el) {
      if (!tooltip || !frame) {
        return;
      }
      var pains = parseInt(el.getAttribute("data-pains") || "0", 10);
      var name = el.getAttribute("data-name") || el.getAttribute("data-country") || "";
      tooltip.textContent = name + " / " + (pains > 0 ? pains + " PAINS" : "NO SIGNAL YET");
      tooltip.style.opacity = "1";
      tooltip.setAttribute("aria-hidden", "false");
    }

    function hideTip() {
      if (!tooltip) {
        return;
      }
      tooltip.style.opacity = "0";
      tooltip.setAttribute("aria-hidden", "true");
    }

    svg.addEventListener("pointerover", function (e) {
      var el = e.target.closest ? e.target.closest(".country-path") : null;
      if (el) {
        showTip(el.querySelector("path") || el);
      }
    });
    svg.addEventListener("pointerout", function (e) {
      if (e.target.closest && e.target.closest(".country-path")) {
        hideTip();
      }
    });
    svg.addEventListener("pointermove", function (e) {
      if (!tooltip || tooltip.style.opacity !== "1" || !frame) {
        return;
      }
      var r = frame.getBoundingClientRect();
      var x = clamp(e.clientX - r.left + 12, 8, r.width - 150);
      var y = clamp(e.clientY - r.top + 12, 8, r.height - 40);
      tooltip.style.left = x + "px";
      tooltip.style.top = y + "px";
    });

    // Selected country: keep the highlight while the side panel swaps. htmx
    // does not process SVG anchors, so the request is issued here; the href
    // stays as the no-JS fallback to the country hub.
    svg.addEventListener("click", function (e) {
      var link = e.target.closest ? e.target.closest(".country-path") : null;
      if (!link) {
        return;
      }
      svg.querySelectorAll(".country-path.is-selected").forEach(function (n) {
        n.classList.remove("is-selected");
      });
      link.classList.add("is-selected");

      var url = link.getAttribute("hx-get");
      if (url && window.htmx && !e.defaultPrevented) {
        e.preventDefault();
        window.htmx.ajax("GET", url, { target: "#map-side", swap: "innerHTML" });
      }
    });

    apply();

    // Region chips live in the same swapped subtree; expose the animation so
    // the delegated handler can drive it without reaching into this closure.
    svg._mapAnimate = animateTo;
  }

  function setActiveRegion(region) {
    document.querySelectorAll("[data-map-region]").forEach(function (btn) {
      var active = btn.getAttribute("data-map-region") === region;
      btn.classList.toggle("chip-active", active);
      btn.setAttribute("aria-pressed", active ? "true" : "false");
    });
  }

  function bindRegions() {
    document.querySelectorAll("[data-map-region]").forEach(function (btn) {
      if (btn.dataset.bound === "1") {
        return;
      }
      btn.dataset.bound = "1";
      btn.addEventListener("click", function () {
        var box = (btn.getAttribute("data-viewbox") || "0 0 880 440").split(" ").map(Number);
        setActiveRegion(btn.getAttribute("data-map-region"));
        var svg = document.getElementById("map-svg");
        if (!svg) {
          return;
        }
        if (svg._mapAnimate) {
          svg._mapAnimate(box);
        }
      });
    });
  }

  document.addEventListener("DOMContentLoaded", function () {
    initMap();
    bindRegions();
  });

  document.addEventListener("htmx:afterSwap", function (e) {
    var t = e.target;
    if (t && t.id === "map-section") {
      initMap();
      bindRegions();
    }
  });
})();
