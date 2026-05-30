<script>
	import { onMount, onDestroy } from "svelte";
	import { createBarcodeDetector } from "$lib/components/scanner/barcode_detector.js";

	let {
		onDecode,
		onStatusChange,
		showControls = true,
		scanning = $bindable(false),
	} = $props();

	/** @type {HTMLVideoElement|null} */
	let videoEl = $state(null);
	/** @type {MediaStream|null} */
	let stream = null;
	/** @type {any} */
	let animFrameId = null;
	/** @type {any} */
	let detector = null;
	let starting = false;
	let lastDecoded = "";
	let lastDecodeTime = 0;

	// Cooldown: gleichen Code nicht doppelt innerhalb von 3 Sekunden melden
	const DECODE_COOLDOWN_MS = 3000;

	onMount(() => {
		setTimeout(startScanner, 300);
	});

	onDestroy(() => {
		stopScanner();
	});

	export async function startScanner() {
		if (scanning || starting) return;
		starting = true;

		try {
			const detectorResult = await createBarcodeDetector();
			if (!detectorResult) {
				onStatusChange(
					"Barcode-Erkennung wird von diesem Browser nicht unterstützt.",
				);
				return;
			}
			detector = detectorResult.detector;

			// Kamera-Stream starten
			stream = await navigator.mediaDevices.getUserMedia({
				video: {
					facingMode: { ideal: "environment" },
					width: { ideal: 1280 },
					height: { ideal: 720 },
				},
				audio: false,
			});

			if (videoEl) {
				videoEl.srcObject = stream;
				await videoEl.play();
			}

			scanning = true;
			onStatusChange("Kamera aktiv. Barcode scannen.");

			// Scan-Schleife starten
			scanLoop();
		} catch (error) {
			console.error("Kamerafehler:", error);
			const errMsg = error instanceof Error ? error.message : String(error);
			onStatusChange(`Kamerafehler: ${errMsg}`);
		} finally {
			starting = false;
		}
	}

	function scanLoop() {
		if (!scanning || !videoEl || !detector) return;

		animFrameId = requestAnimationFrame(async () => {
			if (!scanning || !videoEl || videoEl.readyState < 2) {
				// Video noch nicht bereit, nächsten Frame abwarten
				scanLoop();
				return;
			}

			try {
				const barcodes = await detector.detect(videoEl);
				if (barcodes && barcodes.length > 0) {
					const code = barcodes[0].rawValue;
					const now = Date.now();

					// Cooldown prüfen: gleichen Code nicht doppelt melden
					if (
						code !== lastDecoded ||
						now - lastDecodeTime > DECODE_COOLDOWN_MS
					) {
						lastDecoded = code;
						lastDecodeTime = now;
						onDecode(code);
					}
				}
			} catch {
				// Einzelne Frame-Fehler ignorieren, weiter scannen
			}

			// Nächsten Frame nach kurzer Pause analysieren (~15fps)
			if (scanning) {
				setTimeout(() => scanLoop(), 66);
			}
		});
	}

	export async function stopScanner() {
		scanning = false;

		if (animFrameId) {
			cancelAnimationFrame(animFrameId);
			animFrameId = null;
		}

		if (stream) {
			stream.getTracks().forEach((/** @type {MediaStreamTrack} */ t) => t.stop());
			stream = null;
		}

		if (videoEl) {
			videoEl.srcObject = null;
		}

		detector = null;
	}

	export function restart() {
		stopScanner().then(() => startScanner());
	}
</script>

<div
	class="relative overflow-hidden rounded-2xl border border-slate-200 bg-slate-950 min-h-56 aspect-4/3 w-full max-w-sm mx-auto shadow-inner"
>
	<!-- svelte-ignore element_invalid_self_closing_tag -->
	<video
		bind:this={videoEl}
		autoplay
		playsinline
		muted
		class="absolute inset-0 h-full w-full object-cover"
	/>
	<!-- Scan-Rahmen Overlay -->
	{#if scanning}
		<div
			class="absolute inset-0 flex items-center justify-center pointer-events-none"
		>
			<div
				class="border-2 border-blue-500 rounded-lg opacity-60 animate-pulse"
				style="width: 80%; height: 40%;"
			></div>
		</div>
	{/if}
</div>

{#if showControls}
	<div class="mt-4 flex gap-3 justify-center">
		<button
			onclick={startScanner}
			disabled={scanning}
			class="rounded-xl bg-blue-600 px-5 py-2.5 text-sm font-bold text-white hover:bg-blue-700 disabled:opacity-60 transition-colors cursor-pointer shadow-sm"
			>Starten</button
		>
		<button
			onclick={stopScanner}
			disabled={!scanning}
			class="rounded-xl bg-slate-100 px-5 py-2.5 text-sm font-semibold text-slate-705 hover:bg-slate-200 disabled:opacity-60 transition-colors cursor-pointer"
			>Stoppen</button
		>
	</div>
{/if}
