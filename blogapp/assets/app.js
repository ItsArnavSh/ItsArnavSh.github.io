(function () {
    "use strict";

    const $ = (sel, root = document) => root.querySelector(sel);
    const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

    function formatViews(n) {
        n = Number(n) || 0;
        if (n < 1000) return String(n);
        if (n < 1000000) {
            const v = n / 1000;
            return (v < 10 ? v.toFixed(1) : Math.round(v)) + "k";
        }
        return (n / 1000000).toFixed(1) + "m";
    }

    function renderMath(root) {
        if (!window.katex) return;
        $$(".math[data-tex]", root).forEach((el) => {
            if (el.dataset.rendered === "1") return;
            const tex = el.getAttribute("data-tex");
            if (!tex) return;
            try {
                window.katex.render(tex, el, {
                    displayMode: el.classList.contains("math-display"),
                    throwOnError: false,
                    strict: false,
                    trust: false,
                });
                el.dataset.rendered = "1";
            } catch (e) {
                /* Keep the readable TeX fallback. */
            }
        });
    }

    function initLook() {
        const btn = $("[data-look-toggle]");
        if (!btn) return;
        const label = $("[data-look-label]", btn);
        const apply = (colour) => {
            document.body.classList.toggle("colour-mode", colour);
            btn.setAttribute("aria-pressed", colour ? "true" : "false");
            if (label) label.textContent = colour ? "Colour" : "B&W";
            const meta = document.querySelector('meta[name="theme-color"]');
            if (meta) {
                const dark = document.body.classList.contains("theme-dark");
                meta.setAttribute(
                    "content",
                    colour ? (dark ? "#17120e" : "#f7f1e6") : dark ? "#0f0f0e" : "#f3f3ef"
                );
            }
        };
        let colour = false;
        try {
            const stored =
                localStorage.getItem("arnav-blog-colour") ||
                localStorage.getItem("arnav-blog-photos");
            colour = stored === "colour" || stored === "color";
        } catch (e) {}
        apply(colour);
        btn.addEventListener("click", () => {
            colour = !colour;
            apply(colour);
            try {
                localStorage.setItem("arnav-blog-colour", colour ? "colour" : "mono");
            } catch (e) {}
        });
    }

    function isVisible(el) {
        return !!(el.offsetWidth || el.offsetHeight || el.getClientRects().length);
    }

    function initSearchShortcut() {
        document.addEventListener(
            "keydown",
            (e) => {
                if (
                    !(e.ctrlKey || e.metaKey) ||
                    e.altKey ||
                    e.shiftKey ||
                    String(e.key).toLowerCase() !== "k"
                ) {
                    return;
                }
                const mainSearch = $("#q-main");
                const target =
                    mainSearch && isVisible(mainSearch)
                        ? mainSearch
                        : $$("[data-search-input]").find(isVisible);
                e.preventDefault();
                e.stopPropagation();
                if (!target) {
                    window.location.assign("/search");
                    return;
                }
                target.focus();
                target.select();
            },
            true
        );
    }

    function toast(message) {
        let el = $(".share-toast");
        if (!el) {
            el = document.createElement("div");
            el.className = "share-toast";
            el.setAttribute("role", "status");
            document.body.appendChild(el);
        }
        el.textContent = message;
        el.classList.add("visible");
        clearTimeout(el._timer);
        el._timer = setTimeout(() => el.classList.remove("visible"), 1900);
    }

    async function copyText(text) {
        try {
            await navigator.clipboard.writeText(text);
            return true;
        } catch (e) {
            const ta = document.createElement("textarea");
            ta.value = text;
            ta.style.position = "fixed";
            ta.style.opacity = "0";
            document.body.appendChild(ta);
            ta.select();
            let ok = false;
            try {
                ok = document.execCommand("copy");
            } catch (err) {}
            ta.remove();
            return ok;
        }
    }

    function initShare() {
        document.addEventListener("click", async (e) => {
            const btn = e.target.closest("[data-share]");
            if (!btn) return;
            const mode = btn.getAttribute("data-share");
            const url = btn.getAttribute("data-url") || window.location.href;
            if (mode === "copy") {
                const ok = await copyText(url);
                toast(ok ? "Link copied" : "Could not copy the link");
                btn.classList.add("copied");
                setTimeout(() => btn.classList.remove("copied"), 1400);
            } else if (mode === "native") {
                if (navigator.share) {
                    try {
                        await navigator.share({
                            title: document.title,
                            url: url,
                        });
                    } catch (err) {}
                } else {
                    const ok = await copyText(url);
                    toast(ok ? "Link copied" : "Sharing is not available here");
                }
            }
        });
    }

    function initViews() {
        const body = document.body;
        const id = body.dataset.postId;
        const slug = body.dataset.postSlug;
        const blog = body.dataset.blog;
        const endpoint = body.dataset.viewsUrl;
        if (!id || !slug || !endpoint || body.dataset.print === "1") return;
        if (navigator.doNotTrack === "1" || window.doNotTrack === "1") return;

        fetch(endpoint, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ blog: blog, slug: slug }),
            keepalive: true,
            credentials: "same-origin",
        })
            .then((r) => (r.ok ? r.json() : null))
            .then((data) => {
                if (!data || typeof data.views !== "number") return;
                const value = formatViews(data.views);
                $$("[data-view-count]").forEach((el) => {
                    el.textContent = value;
                });
                $$("[data-view-word]").forEach((el) => {
                    el.textContent = data.views === 1 ? "view" : "views";
                });
                if (data.counted) {
                    const el = $("[data-view-count]");
                    if (el) {
                        el.classList.add("view-bump");
                        setTimeout(() => el.classList.remove("view-bump"), 700);
                    }
                }
            })
            .catch(() => {});
    }

    function initProgress() {
        const bar = $(".reading-progress span");
        const article = $(".article-body");
        if (!bar || !article) return;
        let ticking = false;
        const update = () => {
            ticking = false;
            const rect = article.getBoundingClientRect();
            const top = window.scrollY + rect.top;
            const height = Math.max(1, article.offsetHeight - window.innerHeight * 0.4);
            const pct = Math.min(100, Math.max(0, ((window.scrollY - top + 220) / height) * 100));
            bar.style.width = pct + "%";
        };
        window.addEventListener(
            "scroll",
            () => {
                if (!ticking) {
                    ticking = true;
                    requestAnimationFrame(update);
                }
            },
            { passive: true },
        );
        update();
    }

    function initCodeCopy() {
        document.addEventListener("click", async (e) => {
            const btn = e.target.closest(".code-copy");
            if (!btn) return;
            const block = btn.closest(".code-block");
            const code = block ? $("code", block) : null;
            if (!code) return;
            const ok = await copyText(code.textContent);
            const old = btn.textContent;
            btn.textContent = ok ? "Copied" : "Failed";
            setTimeout(() => {
                btn.textContent = old;
            }, 1400);
        });
    }

    function initSearchSuggest() {
        const form = $(".masthead-search");
        const input = form ? $("input[data-search-input]", form) : null;
        if (!form || !input) return;
        const box = document.createElement("div");
        box.className = "search-suggest";
        box.hidden = true;
        form.appendChild(box);

        let timer = null;
        let controller = null;
        let active = -1;

        const hide = () => {
            box.hidden = true;
            active = -1;
        };

        const draw = (results) => {
            if (!results.length) {
                hide();
                return;
            }
            box.innerHTML = results
                .map(
                    (r) =>
                        `<a href="${r.url}"><span>${r.blog}</span><b>${escapeHTML(r.title)}</b><small>${r.date}</small></a>`,
                )
                .join("");
            box.hidden = false;
        };

        const load = () => {
            const q = input.value.trim();
            if (q.length < 2) {
                hide();
                return;
            }
            if (controller) controller.abort();
            controller = new AbortController();
            fetch("/api/search?q=" + encodeURIComponent(q) + "&limit=6", {
                signal: controller.signal,
                headers: { Accept: "application/json" },
            })
                .then((r) => (r.ok ? r.json() : null))
                .then((data) => data && draw(data.results || []))
                .catch(() => {});
        };

        input.addEventListener("input", () => {
            clearTimeout(timer);
            timer = setTimeout(load, 220);
        });
        input.addEventListener("focus", () => {
            if (input.value.trim().length >= 2) load();
        });
        input.addEventListener("keydown", (e) => {
            if (box.hidden) return;
            const links = $$("a", box);
            if (e.key === "Escape") {
                hide();
                input.blur();
            } else if (e.key === "ArrowDown") {
                e.preventDefault();
                active = Math.min(links.length - 1, active + 1);
                links.forEach((a, i) => a.classList.toggle("active", i === active));
            } else if (e.key === "ArrowUp") {
                e.preventDefault();
                active = Math.max(0, active - 1);
                links.forEach((a, i) => a.classList.toggle("active", i === active));
            } else if (e.key === "Enter" && active >= 0 && links[active]) {
                e.preventDefault();
                links[active].click();
            }
        });
        document.addEventListener("click", (e) => {
            if (!form.contains(e.target)) hide();
        });
    }

    function escapeHTML(s) {
        return String(s).replace(/[&<>"']/g, (c) => {
            switch (c) {
                case "&":
                    return "&amp;";
                case "<":
                    return "&lt;";
                case ">":
                    return "&gt;";
                case '"':
                    return "&quot;";
                default:
                    return "&#39;";
            }
        });
    }

    document.addEventListener("DOMContentLoaded", () => {
        initLook();
        renderMath(document);
        initShare();
        initViews();
        initProgress();
        initCodeCopy();
        initSearchSuggest();
        initSearchShortcut();
        if (document.body.dataset.print === "1") {
            setTimeout(() => window.print(), 650);
        }
    });
})();
