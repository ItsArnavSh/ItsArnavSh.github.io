(function () {
    "use strict";

    const $ = (sel, root = document) => root.querySelector(sel);
    const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));
    const csrf = document.body.dataset.csrf || "";

    function toast(message) {
        let el = $(".admin-toast");
        if (!el) {
            el = document.createElement("div");
            el.className = "admin-toast";
            el.setAttribute("role", "status");
            document.body.appendChild(el);
        }
        el.textContent = message;
        el.classList.add("visible");
        clearTimeout(el._timer);
        el._timer = setTimeout(() => el.classList.remove("visible"), 2400);
    }

    function slugify(value) {
        return String(value)
            .toLowerCase()
            .trim()
            .replace(/['’"]/g, "")
            .replace(/[^a-z0-9\u00c0-\u024f]+/g, "-")
            .replace(/^-+|-+$/g, "")
            .replace(/-{2,}/g, "-")
            .slice(0, 90);
    }

    function stripMarkdown(md) {
        return String(md)
            .replace(/```[\s\S]*?```/g, " ")
            .replace(/!\[([^\]]*)\]\([^)]*\)/g, "$1")
            .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
            .replace(/```[\s\S]*?```/g, " ")
            .replace(/<[^>]+>/g, " ")
            .replace(/[#>*_~`\-]+/g, " ")
            .replace(/\s+/g, " ")
            .trim();
    }

    function wordCount(md) {
        const text = stripMarkdown(md);
        return text ? text.split(/\s+/).length : 0;
    }

    function renderMath(root) {
        if (!window.katex || !root) return;
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
                /* Keep the TeX fallback. */
            }
        });
    }

    /* ------------------------------------------------------------
       GitHub device flow
       ------------------------------------------------------------ */
    function initDeviceFlow() {
        const card = $(".device-card[data-device-poll]");
        if (!card) return;
        const endpoint = card.dataset.devicePoll;
        const statusEl = $("[data-device-status]", card);
        const timerEl = $("[data-device-timer]", card);
        const checkBtn = $("[data-device-check]", card);
        const interval = Math.max(5, Number(card.dataset.interval) || 5) * 1000;
        let remaining = Number(card.dataset.expires) || 900;
        let stopped = false;

        const setStatus = (text, kind) => {
            if (statusEl) {
                statusEl.textContent = text;
                statusEl.classList.toggle("error", kind === "error");
                statusEl.classList.toggle("ok", kind === "ok");
            }
        };

        const countdown = setInterval(() => {
            remaining -= 1;
            if (timerEl) timerEl.textContent = Math.max(0, remaining);
            if (remaining <= 0) {
                clearInterval(countdown);
                setStatus("The code expired. Reload to start again.", "error");
                stopped = true;
            }
        }, 1000);

        const poll = async () => {
            if (stopped) return;
            try {
                const res = await fetch(endpoint, {
                    headers: { Accept: "application/json" },
                    credentials: "same-origin",
                    cache: "no-store",
                });
                const data = await res.json();
                if (data.status === "ok") {
                    stopped = true;
                    clearInterval(countdown);
                    setStatus("Approved. Opening the desk…", "ok");
                    window.location.href = data.redirect || "/admin";
                    return;
                }
                if (data.status === "error") {
                    stopped = true;
                    clearInterval(countdown);
                    setStatus(data.message || "GitHub rejected the sign-in.", "error");
                    return;
                }
                setStatus(data.message || "Waiting for GitHub…");
            } catch (e) {
                setStatus("Could not reach the desk. Retrying…");
            }
            if (!stopped) setTimeout(poll, interval);
        };

        if (checkBtn) {
            checkBtn.addEventListener("click", () => {
                if (stopped) return;
                setStatus("Checking with GitHub…");
                poll();
            });
        }
        setTimeout(poll, Math.min(interval, 4000));
    }

    /* ------------------------------------------------------------
       Editor
       ------------------------------------------------------------ */
    function initEditor() {
        const form = $("#post-form");
        if (!form) return;

        const title = $("#title");
        const slug = $("[data-slug-input]") || $("#slug");
        const body = $("[data-md-input]") || $("#body_md");
        const preview = $("[data-preview]");
        const split = $("[data-editor-split]");
        const status = $("[data-save-status]");
        const wordEl = $("[data-word-count]");
        const coverInput = $("[data-cover-input]") || $("#cover_image");
        const coverPreview = $("[data-cover-preview]");
        const coverUpload = $("[data-cover-upload]");
        const bodyUpload = $("[data-body-upload]");
        const mediaUpload = $("[data-media-upload]");
        const mediaGrid = $("[data-media-grid]");
        const mediaEmpty = $("[data-media-empty]");
        const deleteForm = $("form[data-confirm-form]");
        const draftKey = "arnav-blog-draft:" + window.location.pathname;
        let slugTouched = Boolean(slug && slug.value.trim());
        let previewTimer = null;
        let previewController = null;
        let draftTimer = null;

        flagUnsaved(false);

        function flagUnsaved(dirty) {
            if (!status) return;
            if (dirty) {
                status.textContent = "Unsaved changes";
                status.classList.add("dirty");
            } else {
                status.textContent = "All changes saved";
                status.classList.remove("dirty");
            }
        }

        /* Slug generation */
        if (title && slug) {
            title.addEventListener("input", () => {
                if (!slugTouched) slug.value = slugify(title.value);
            });
            slug.addEventListener("input", () => {
                slugTouched = slug.value.trim() !== "";
                if (slug.value && /[^a-z0-9-]/.test(slug.value)) {
                    const at = slug.selectionStart;
                    slug.value = slugify(slug.value);
                    try {
                        slug.setSelectionRange(at, at);
                    } catch (e) {}
                }
            });
        }

        /* Word count + draft state */
        const updateWords = () => {
            if (!body || !wordEl) return;
            const n = wordCount(body.value);
            wordEl.textContent = n + (n === 1 ? " word" : " words");
        };
        updateWords();

        const snapshot = () => ({
            title: title ? title.value : "",
            subtitle: ($("#subtitle") || {}).value || "",
            excerpt: ($("#excerpt") || {}).value || "",
            body: body ? body.value : "",
            cover: coverInput ? coverInput.value : "",
            savedAt: Date.now(),
        });

        const saveDraft = () => {
            if (!body) return;
            try {
                const data = snapshot();
                if (!data.body.trim() && !data.title.trim()) {
                    localStorage.removeItem(draftKey);
                    return;
                }
                localStorage.setItem(draftKey, JSON.stringify(data));
                if (status) {
                    status.textContent = "Local draft saved";
                    status.classList.remove("dirty");
                }
            } catch (e) {}
        };

        const scheduleDraft = () => {
            flagUnsaved(true);
            clearTimeout(draftTimer);
            draftTimer = setTimeout(saveDraft, 900);
        };

        const fields = [title, $("#subtitle"), $("#excerpt"), body, coverInput].filter(Boolean);
        fields.forEach((el) => {
            el.addEventListener("input", () => {
                updateWords();
                scheduleDraft();
                if (el === coverInput) updateCoverPreview();
            });
        });

        /* Restore a local draft if it is genuinely newer */
        try {
            const raw = localStorage.getItem(draftKey);
            if (raw) {
                const saved = JSON.parse(raw);
                const current = {
                    title: title ? title.value : "",
                    body: body ? body.value : "",
                };
                if (saved && (saved.body || saved.title) && (saved.body !== current.body || saved.title !== current.title)) {
                    const statusRow = $(".editor-status");
                    if (statusRow) {
                        const btn = document.createElement("button");
                        btn.type = "button";
                        btn.className = "link-button";
                        btn.textContent = "Restore unsaved draft";
                        btn.addEventListener("click", () => {
                            if (title) title.value = saved.title || "";
                            if ($("#subtitle")) $("#subtitle").value = saved.subtitle || "";
                            if ($("#excerpt")) $("#excerpt").value = saved.excerpt || "";
                            if (body) body.value = saved.body || "";
                            if (coverInput) coverInput.value = saved.cover || "";
                            updateWords();
                            updateCoverPreview();
                            refreshPreview();
                            btn.remove();
                            toast("Local draft restored");
                        });
                        statusRow.appendChild(btn);
                    }
                }
            }
        } catch (e) {}

        form.addEventListener("submit", () => {
            try {
                localStorage.removeItem(draftKey);
            } catch (e) {}
        });

        /* Cover preview */
        function updateCoverPreview() {
            if (!coverPreview) return;
            const url = coverInput ? coverInput.value.trim() : "";
            if (!url) {
                coverPreview.classList.add("empty");
                coverPreview.innerHTML = "<span>No cover selected</span>";
                return;
            }
            coverPreview.classList.remove("empty");
            coverPreview.innerHTML = "";
            const img = document.createElement("img");
            img.src = url;
            img.alt = "";
            img.onerror = () => {
                coverPreview.classList.add("empty");
                coverPreview.innerHTML = "<span>Image not found</span>";
            };
            coverPreview.appendChild(img);
        }

        /* Preview */
        const refreshPreview = () => {
            if (!body || !preview) return;
            clearTimeout(previewTimer);
            previewTimer = setTimeout(async () => {
                if (previewController) previewController.abort();
                previewController = new AbortController();
                try {
                    const res = await fetch(form.dataset.previewUrl, {
                        method: "POST",
                        headers: {
                            "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8",
                            Accept: "application/json",
                            "X-CSRF-Token": csrf,
                        },
                        body: new URLSearchParams({ markdown: body.value, csrf: csrf }),
                        signal: previewController.signal,
                        credentials: "same-origin",
                    });
                    if (!res.ok) return;
                    const data = await res.json();
                    preview.innerHTML = data.html || '<div class="preview-placeholder">Nothing to preview yet.</div>';
                    renderMath(preview);
                    if (wordEl && typeof data.words === "number") {
                        wordEl.textContent = data.words + (data.words === 1 ? " word" : " words");
                    }
                } catch (e) {
                    if (e.name !== "AbortError") {
                        /* Keep the last good preview. */
                    }
                }
            }, 420);
        };
        if (body) body.addEventListener("input", refreshPreview);
        if (body && body.value.trim()) refreshPreview();

        /* View modes */
        $$("[data-view]").forEach((btn) => {
            btn.addEventListener("click", () => {
                const mode = btn.dataset.view;
                $$("[data-view]").forEach((b) => b.classList.toggle("active", b === btn));
                if (split) {
                    split.classList.remove("mode-write", "mode-preview");
                    if (mode === "write") split.classList.add("mode-write");
                    if (mode === "preview") split.classList.add("mode-preview");
                }
                try {
                    localStorage.setItem("arnav-blog-view", mode);
                } catch (e) {}
                if (mode !== "write") refreshPreview();
            });
        });
        try {
            const savedMode = localStorage.getItem("arnav-blog-view");
            if (savedMode === "write" || savedMode === "preview" || savedMode === "split") {
                const btn = $(`[data-view="${savedMode}"]`);
                if (btn) btn.click();
            } else if (split) {
                split.classList.remove("mode-write", "mode-preview");
            }
        } catch (e) {}

        /* Toolbar */
        function insertText(text) {
            if (!body) return;
            const start = body.selectionStart;
            const end = body.selectionEnd;
            body.setRangeText(text, start, end, "end");
            body.focus();
            body.dispatchEvent(new Event("input"));
        }

        function surround(before, after, placeholder) {
            if (!body) return;
            const start = body.selectionStart;
            const end = body.selectionEnd;
            const selected = body.value.slice(start, end) || placeholder || "";
            body.setRangeText(before + selected + after, start, end, "end");
            body.focus();
            body.dispatchEvent(new Event("input"));
        }

        function prefixLines(prefix) {
            if (!body) return;
            const start = body.selectionStart;
            const end = body.selectionEnd;
            const lineStart = body.value.lastIndexOf("\n", start - 1) + 1;
            const segment = body.value.slice(lineStart, end) || "";
            const lines = segment.split("\n");
            const replaced = lines.map((line) => prefix + line).join("\n");
            body.setRangeText(replaced, lineStart, end, "end");
            body.focus();
            body.dispatchEvent(new Event("input"));
        }

        $$("[data-md]").forEach((btn) => {
            btn.addEventListener("click", () => {
                if (!body) return;
                switch (btn.dataset.md) {
                    case "h2":
                        prefixLines("## ");
                        break;
                    case "bold":
                        surround("**", "**", "bold text");
                        break;
                    case "italic":
                        surround("*", "*", "italic text");
                        break;
                    case "quote":
                        prefixLines("> ");
                        break;
                    case "link":
                        surround("[", "](https://)", "link text");
                        break;
                    case "image":
                        surround("![", "](/media/)", "alt text");
                        break;
                    case "code":
                        surround("\n```go\n", "\n```\n", "code");
                        break;
                    case "math":
                        surround("\n$$\n", "\n$$\n", "x^2 + y^2 = z^2");
                        break;
                    case "list":
                        prefixLines("- ");
                        break;
                    case "rule":
                        surround("\n\n---\n\n", "", "");
                        break;
                }
            });
        });

        /* Uploads */
        async function uploadFile(file) {
            if (!file) return null;
            if (file.size > 12 * 1024 * 1024) {
                toast("That image is larger than 12 MB");
                return null;
            }
            const fd = new FormData();
            fd.append("file", file);
            const res = await fetch("/admin/upload", {
                method: "POST",
                body: fd,
                headers: { Accept: "application/json", "X-CSRF-Token": csrf },
                credentials: "same-origin",
            });
            const data = await res.json().catch(() => null);
            if (!res.ok || !data || !data.url) {
                toast((data && data.error) || "Upload failed");
                return null;
            }
            return data;
        }

        /* Media library */
        const mediaItems = [];

        function prettyMediaAlt(name) {
            return String(name || "image")
                .replace(/\.[^.]+$/, "")
                .replace(/^\d{8}-\d{6}-/, "")
                .replace(/-[0-9a-f]{6}$/, "")
                .replace(/[-_]+/g, " ")
                .trim() || "image";
        }

        function rememberMedia(data, session, alt) {
            if (!data || !data.url || !mediaGrid) return;
            let item = mediaItems.find((m) => m.url === data.url);
            if (!item) {
                item = {
                    url: data.url,
                    name: data.name || data.url.split("/").pop(),
                    alt: alt || prettyMediaAlt(data.name || data.url),
                    session: Boolean(session),
                };
                mediaItems.push(item);
            } else {
                if (alt) item.alt = alt;
                item.session = item.session || Boolean(session);
            }
            if (session) {
                const at = mediaItems.indexOf(item);
                if (at > 0) {
                    mediaItems.splice(at, 1);
                    mediaItems.unshift(item);
                }
            }
            renderMedia();
        }

        function renderMedia() {
            if (!mediaGrid) return;
            mediaGrid.innerHTML = "";
            if (!mediaItems.length) {
                if (mediaEmpty) {
                    mediaEmpty.hidden = false;
                    mediaGrid.appendChild(mediaEmpty);
                } else {
                    const empty = document.createElement("p");
                    empty.className = "media-empty";
                    empty.textContent = "No images yet. Upload one here, or drop an image anywhere in the editor.";
                    mediaGrid.appendChild(empty);
                }
                return;
            }
            mediaItems.forEach((item) => {
                const btn = document.createElement("button");
                btn.type = "button";
                btn.className = "media-item";
                btn.title = "Insert " + item.alt;
                btn.setAttribute("aria-label", "Insert image: " + item.alt);
                const img = document.createElement("img");
                img.src = item.url;
                img.alt = "";
                img.loading = "lazy";
                img.addEventListener("error", () => {
                    btn.classList.add("broken");
                    btn.title = item.name + " (image failed to load)";
                });
                btn.appendChild(img);
                btn.addEventListener("click", () => {
                    if (!body) return;
                    insertText("\n![" + item.alt + "](" + item.url + ")\n");
                    toast("Image inserted");
                });
                mediaGrid.appendChild(btn);
            });
        }

        async function loadMediaLibrary() {
            if (!mediaGrid) return;
            try {
                const res = await fetch("/admin/media", {
                    headers: { Accept: "application/json" },
                    credentials: "same-origin",
                    cache: "no-store",
                });
                if (!res.ok) return;
                const data = await res.json();
                (data.items || []).forEach((item) => {
                    if (item && item.url) rememberMedia(item, false);
                });
            } catch (e) {
                /* The library is a convenience; the editor still works without it. */
            }
        }

        if (mediaUpload) {
            mediaUpload.addEventListener("change", async () => {
                const files = Array.from(mediaUpload.files || []);
                if (!files.length) return;
                toast(files.length === 1 ? "Uploading image…" : "Uploading " + files.length + " images…");
                let uploaded = 0;
                for (const file of files) {
                    const data = await uploadFile(file);
                    if (data) {
                        rememberMedia(data, true);
                        uploaded += 1;
                    }
                }
                if (uploaded) {
                    toast(uploaded === 1 ? "Uploaded — click the thumbnail to insert" : "Uploaded " + uploaded + " images — click one to insert");
                }
                mediaUpload.value = "";
            });
        }
        loadMediaLibrary();

        if (coverUpload) {
            coverUpload.addEventListener("change", async () => {
                const file = coverUpload.files && coverUpload.files[0];
                if (!file) return;
                toast("Uploading cover…");
                const data = await uploadFile(file);
                if (data) {
                    if (coverInput) coverInput.value = data.url;
                    rememberMedia(data, true);
                    updateCoverPreview();
                    scheduleDraft();
                    toast("Cover uploaded");
                }
                coverUpload.value = "";
            });
        }

        if (bodyUpload) {
            bodyUpload.addEventListener("change", async () => {
                const file = bodyUpload.files && bodyUpload.files[0];
                if (!file) return;
                toast("Uploading image…");
                const data = await uploadFile(file);
                if (data && body) {
                    const alt = (file.name || "image").replace(/\.[^.]+$/, "").replace(/[-_]+/g, " ");
                    rememberMedia(data, true, alt);
                    insertText("\n![" + alt + "](" + data.url + ")\n");
                    toast("Image inserted");
                }
                bodyUpload.value = "";
            });
        }

        /* Drag and drop images anywhere in the editor */
        ["dragenter", "dragover"].forEach((name) => {
            form.addEventListener(name, (e) => {
                if (e.dataTransfer && Array.from(e.dataTransfer.types || []).includes("Files")) {
                    e.preventDefault();
                    form.classList.add("is-dragging");
                }
            });
        });
        ["dragleave", "drop"].forEach((name) => {
            form.addEventListener(name, (e) => {
                if (name === "dragleave" && e.target !== form) return;
                form.classList.remove("is-dragging");
            });
        });
        form.addEventListener("drop", async (e) => {
            const files = e.dataTransfer ? Array.from(e.dataTransfer.files || []).filter((f) => f.type.startsWith("image/")) : [];
            if (!files.length) return;
            e.preventDefault();
            for (const file of files) {
                toast("Uploading " + file.name + "…");
                const data = await uploadFile(file);
                if (data && body) {
                    const alt = (file.name || "image").replace(/\.[^.]+$/, "").replace(/[-_]+/g, " ");
                    rememberMedia(data, true, alt);
                    insertText("\n![" + alt + "](" + data.url + ")\n");
                }
            }
        });

        /* Delete confirmation */
        if (deleteForm) {
            deleteForm.addEventListener("submit", (e) => {
                e.preventDefault();
                const btn = deleteForm.querySelector("[data-confirm]");
                const message = (btn && btn.dataset.confirm) || "Delete this dispatch permanently?";
                if (window.confirm(message)) deleteForm.submit();
            });
        }

        /* Keyboard shortcuts */
        form.addEventListener("keydown", (e) => {
            const meta = e.metaKey || e.ctrlKey;
            if (!meta) return;
            if (e.key === "s") {
                e.preventDefault();
                form.requestSubmit ? form.requestSubmit() : form.submit();
            } else if (e.key === "b") {
                e.preventDefault();
                surround("**", "**", "bold text");
            } else if (e.key === "i") {
                e.preventDefault();
                surround("*", "*", "italic text");
            }
        });
    }

    document.addEventListener("DOMContentLoaded", () => {
        initDeviceFlow();
        initEditor();
    });
})();
