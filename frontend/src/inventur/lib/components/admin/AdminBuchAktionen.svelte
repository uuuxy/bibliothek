<script>
	import { apiFetch } from '../../../../lib/apiFetch.js';
    import { appState, showToast } from "$lib/store.svelte.js";

    let {
        books = $bindable(),
        isEditMode = $bindable(),
        formular = $bindable(),
    } = $props();

    export async function saveChanges() {
        if (!formular.title || !formular.isbn) {
            showToast("Titel und ISBN sind Pflichtfelder", "error");
            return;
        }

        if (formular.id) {
            const originalBook = books.find((/** @type {any} */ b) => b.id === formular.id);
            if (originalBook && Number(formular.stock) < Number(originalBook.stock)) {
                const proceed = confirm(
                    "Achtung: Du verringerst den Gesamtbestand manuell. Das System wird die entsprechende Anzahl an Exemplaren im Hintergrund als verloren markieren. Möchtest du fortfahren?"
                );
                if (!proceed) return;
            }
        }

        try {
            const url = formular.id
                ? `/api/books/${formular.id}`
                : `/api/books`;
            const res = await apiFetch(url, {
                method: formular.id ? "PUT" : "POST",
                credentials: "include",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({
                    ...formular,
                    gradeLevel: Number(formular.gradeLevel),
                    stock: Number(formular.stock),
                    lastCounted: formular.lastCounted || null,
                }),
            });
            if (!res.ok) {
                let errMsg = "Speichern fehlgeschlagen";
                const errData = await res.json().catch(() => null);
                if (errData) {
                    errMsg = errData.error || errData.message || errMsg;
                }
                throw new Error(errMsg);
            }
            const updated = (await res.json()).data;
            books = formular.id
                ? books.map((/** @type {any} */ b) => (b.id === updated.id ? updated : b))
                : [updated, ...books];
            
            // Sync with appState so Omnibox/Catalog update immediately
            if (appState.selectedBook && appState.selectedBook.id === updated.id) {
                appState.selectedBook = updated;
            }

            if (isEditMode) {
                isEditMode = false;
            }
            showToast("Buch erfolgreich gespeichert!", "success");
        } catch (e) {
            showToast(e instanceof Error ? e.message : String(e), "error");
        }
    }

    /** @param {File} file */
    async function compressImageToWebp(file) {
        return new Promise((resolve, reject) => {
            const img = new Image();
            img.onload = () => {
                URL.revokeObjectURL(img.src);
                const canvas = document.createElement("canvas");
                let width = img.width;
                let height = img.height;
                const MAX_WIDTH = 600;
                const MAX_HEIGHT = 900;

                if (width > MAX_WIDTH || height > MAX_HEIGHT) {
                    const ratio = Math.min(
                        MAX_WIDTH / width,
                        MAX_HEIGHT / height,
                    );
                    width = Math.round(width * ratio);
                    height = Math.round(height * ratio);
                }

                canvas.width = width;
                canvas.height = height;
                const ctx = canvas.getContext("2d");
                if (ctx) ctx.drawImage(img, 0, 0, width, height);

                canvas.toBlob(
                    (blob) => {
                        if (!blob) reject(new Error("Compression failed"));
                        else
                            resolve(
                                new File(
                                    [blob],
                                    file.name.replace(/\.[^/.]+$/, ".webp"),
                                    { type: "image/webp" },
                                ),
                            );
                    },
                    "image/webp",
                    0.82,
                );
            };
            img.onerror = () => reject(new Error("Invalid image"));
            img.src = URL.createObjectURL(file);
        });
    }

    /** @param {Event} e */
    export async function handleCoverUpload(e) {
        const target = /** @type {HTMLInputElement} */ (e.target);
        let file = target.files ? target.files[0] : null;
        if (!file || !formular.id) return;

        try {
            if (file.type.startsWith("image/")) {
                file = await compressImageToWebp(file);
            }
        } catch (err) {
            console.error("WebP compression failed, using original file:", err);
        }

        const fd = new FormData();
        fd.append("cover", /** @type {File} */ (file));
        try {
            const res = await apiFetch(`/api/books/${formular.id}/cover-upload`, {
                method: "POST",
                credentials: "include",
                headers: {
                },
                body: fd,
            });
            if (!res.ok) {
                let message = "Upload fehlgeschlagen";
                try {
                    const errorJson = await res.json();
                    if (errorJson?.message) {
                        message = errorJson.message;
                    } else if (errorJson?.error) {
                        message = errorJson.error;
                    }
                } catch {
                    // fallback to default message
                }
                throw new Error(message);
            }
            const json = await res.json();
            formular.coverUrl = json.data.coverUrl;
            books = books.map((/** @type {any} */ b) =>
                b.id === formular.id
                    ? { ...b, coverUrl: json.data.coverUrl }
                    : b,
            );
            showToast("Cover erfolgreich hochgeladen", "success");
        } catch (err) {
            showToast(err instanceof Error ? err.message : String(err), "error");
        }
    }
</script>
