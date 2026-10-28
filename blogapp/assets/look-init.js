(function () {
    try {
        var stored =
            localStorage.getItem("arnav-blog-colour") ||
            localStorage.getItem("arnav-blog-photos");
        var colour = stored === "colour" || stored === "color";
        if (colour) document.body.classList.add("colour-mode");
        var meta = document.querySelector('meta[name="theme-color"]');
        var dark = document.body.classList.contains("theme-dark");
        if (meta) {
            meta.setAttribute(
                "content",
                colour ? (dark ? "#17120e" : "#f7f1e6") : dark ? "#0f0f0e" : "#f3f3ef"
            );
        }
    } catch (e) {}
})();
