// Progressive enhancement: toggles the light/dark theme and keeps the toggle's
// label and aria-pressed in sync. The document head sets the initial theme.
(function () {
  function apply(theme) {
    var dark = theme === "dark";
    document.documentElement.classList.toggle("dark", dark);
    var btn = document.querySelector("[data-theme-toggle]");
    if (btn) {
      btn.setAttribute("aria-pressed", String(dark));
      btn.textContent = dark ? "Light mode" : "Dark mode";
    }
  }

  document.addEventListener("click", function (event) {
    var toggle = event.target.closest ? event.target.closest("[data-theme-toggle]") : null;
    if (!toggle) {
      return;
    }
    var next = document.documentElement.classList.contains("dark") ? "light" : "dark";
    try {
      localStorage.setItem("theme", next);
    } catch (e) {}
    apply(next);
  });

  var stored = null;
  try {
    stored = localStorage.getItem("theme");
  } catch (e) {}
  var prefersDark = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches;
  apply(stored || (prefersDark ? "dark" : "light"));
})();
