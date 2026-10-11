// Applies the saved or system theme before first paint to avoid a flash of the
// wrong colour scheme. Loaded render-blocking from the document head.
(function () {
  try {
    var stored = localStorage.getItem("theme");
    var dark = stored ? stored === "dark" : window.matchMedia("(prefers-color-scheme: dark)").matches;
    document.documentElement.classList.toggle("dark", dark);
  } catch (e) {}
})();
