<script>
    let { onDecode, onStatusChange, disabled } = $props();

    /**
     * @param {File} file
     * @returns {Promise<File>}
     */
    async function downscaleImage(file) {
        // Downscale image first to massively improve barcode recognition on high-res phones (e.g. iPhone 15)
        return new Promise((resolve, reject) => {
            const img = new Image();
            img.onload = () => {
                const canvas = document.createElement("canvas");
                let { width, height } = img;
                const MAX_DIM = 1200; // Optimum für Barcode-Erkennung per ZXing (höher = langsamer, Timeout-Gefahr)

                if (width > MAX_DIM || height > MAX_DIM) {
                    const ratio = Math.min(
                        MAX_DIM / width,
                        MAX_DIM / height,
                    );
                    width = Math.round(width * ratio);
                    height = Math.round(height * ratio);
                }

                canvas.width = width;
                canvas.height = height;
                const ctx = canvas.getContext("2d");

                if (ctx) {
                    // Increase contrast and make black/white for better line detection
                    ctx.filter = "contrast(1.2) grayscale(1)";
                    ctx.drawImage(img, 0, 0, width, height);
                }

                canvas.toBlob(
                    (blob) => {
                        if (blob) {
                            resolve(
                                new File([blob], "scan.jpg", {
                                    type: "image/jpeg",
                                }),
                            );
                        } else {
                            reject(new Error("Canvas toBlob failed"));
                        }
                    },
                    "image/jpeg",
                    0.9,
                );
                URL.revokeObjectURL(img.src);
            };
            img.onerror = () =>
                reject(new Error("Bild konnte nicht geladen werden"));
            img.src = URL.createObjectURL(file);
        });
    }

    /**
     * @param {File} downscaledFile
     * @returns {Promise<string | null>}
     */
    async function scanWithNativeDetector(downscaledFile) {
        // Methode 1: Native BarcodeDetector API (Apple/Android)
        // Dies ist um Welten besser und robuster als ZXing in JS.
        if (!("BarcodeDetector" in window)) return null;

        try {
            // @ts-ignore
            const detector = new window.BarcodeDetector({
                formats: [
                    "ean_13",
                    "ean_8",
                    "upc_a",
                    "upc_e",
                    "code_128",
                ],
            });

            // Bild laden für den Detektor
            const img = new Image();
            const objectUrl = URL.createObjectURL(downscaledFile);

            await new Promise((resolve, reject) => {
                img.onload = resolve;
                img.onerror = reject;
                img.src = objectUrl;
            });

            const barcodes = await detector.detect(img);
            URL.revokeObjectURL(objectUrl);

            if (barcodes.length > 0) {
                return barcodes[0].rawValue;
            }
        } catch (e) {
            console.warn(
                "Nativer BarcodeDetector fehlgeschlagen, probiere Fallback",
                e,
            );
        }
        return null;
    }

    /**
     * @param {File} downscaledFile
     * @returns {Promise<string>}
     */
    async function scanWithHtml5Qrcode(downscaledFile) {
        // Methode 2: Fallback auf html5-qrcode (ZXing in JS)
        const module = await import("html5-qrcode");
        const Html5Qrcode = module.Html5Qrcode;
        const Html5QrcodeSupportedFormats = module.Html5QrcodeSupportedFormats;

        const tempScanner = new Html5Qrcode("hidden-file-scanner", {
            formatsToSupport: [
                Html5QrcodeSupportedFormats.EAN_13,
                Html5QrcodeSupportedFormats.EAN_8,
                Html5QrcodeSupportedFormats.UPC_A,
                Html5QrcodeSupportedFormats.UPC_E,
                Html5QrcodeSupportedFormats.CODE_128,
            ],
            verbose: false,
        });
        const result = await tempScanner.scanFileV2(downscaledFile, true);
        tempScanner.clear();

        return result.decodedText;
    }

    /**
     * @param {Event & { currentTarget: HTMLInputElement }} event
     */
    async function handleFileUpload(event) {
        const files = event.currentTarget.files;
        const file = files ? files[0] : null;
        if (!file) return;

        onStatusChange("Analysiere Foto...", true);

        try {
            const downscaledFile = await downscaleImage(file);

            let decodedText = await scanWithNativeDetector(downscaledFile);

            if (decodedText) {
                onDecode(decodedText);
                return;
            }

            decodedText = await scanWithHtml5Qrcode(downscaledFile);
            onDecode(decodedText);
        } catch (err) {
            console.error("File scan error:", err);
            onStatusChange(
                "Kein Barcode auf dem Foto gefunden. Bitte Bild zuschneiden oder schärfer fotografieren.",
                false,
            );

            try {
                const module = await import("html5-qrcode");
                const Html5Qrcode = module.Html5Qrcode;
                new Html5Qrcode("hidden-file-scanner").clear();
            } catch {
                void 0;
            }
        } finally {
            event.currentTarget.value = "";
        }
    }
</script>

<div class="mt-6 border-t border-zinc-800/80 pt-4">
    <span
        class="block text-xs font-semibold uppercase tracking-wider text-zinc-400 mb-2"
        >Alternativ: Foto vom Barcode machen</span
    >
    <label
        class="cursor-pointer w-full rounded-xl border-2 border-dashed border-zinc-800 bg-zinc-950/40 hover:bg-zinc-850/40 transition-colors px-5 py-4 flex flex-col items-center justify-center gap-2 disabled:opacity-60"
    >
        <svg
            xmlns="http://www.w3.org/2000/svg"
            class="h-8 w-8 text-emerald-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
        >
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z"
            />
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M15 13a3 3 0 11-6 0 3 3 0 016 0z"
            />
        </svg>
        <span class="text-sm font-semibold text-zinc-300"
            >Kamera öffnen / Foto auswählen</span
        >
        <input
            type="file"
            accept="image/*"
            capture="environment"
            class="hidden"
            onchange={handleFileUpload}
            {disabled}
        />
    </label>
</div>

<div
    id="hidden-file-scanner"
    style="position: absolute; top: -9999px; left: -9999px;"
></div>
