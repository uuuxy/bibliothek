<!-- @component WebcamCapture — Aufnahme des Passbilds mit der Webcam.

     Ein Dialog des Hauses (Modal). Dunkel ist nur der Sucher: Ein Kamerabild steht auf Schwarz,
     die Hilfslinien gehören zum Bild. Der erste Fokus liegt auf „Schließen" im Kopf und nicht auf
     der Aufnahme: An der Theke tippt der Handscanner blind und endet mit Enter, das nähme sonst
     ein Foto auf und ersetzte das vorhandene. -->
<script>
	import { AlertTriangle, Camera } from '@lucide/svelte';
	import { apiClient } from './apiFetch.js';
	import Modal from './Modal.svelte';
	import Button from './components/ui/Button.svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import { onMount } from 'svelte';
	import { ruhtWennTraege } from './actions/ruhtWennTraege.js';

	/** @type {{ studentId: string, onCapture: (url: string) => void, onClose: () => void }} */
	let { studentId, onCapture, onClose } = $props();

	/** @type {HTMLVideoElement | null} */
	let videoEl = $state(null);
	/** @type {MediaStream | null} */
	let stream = $state(null);
	/** @type {string | null} */
	let errorMsg = $state(null);
	let isCapturing = $state(false);

	async function startCamera() {
		try {
			errorMsg = null;
			stream = await navigator.mediaDevices.getUserMedia({
				video: {
					width: { ideal: 1920 }, // High resolution for sharp details
					height: { ideal: 1080 },
					aspectRatio: { ideal: 1.7777777778 }, // Widescreen native
					facingMode: 'user'
				}
			});
			if (videoEl) {
				videoEl.srcObject = stream;
			}
		} catch (err) {
			const error = /** @type {any} */ (err);
			errorMsg = 'Kamera-Zugriff fehlgeschlagen: ' + error.message;
		}
	}

	async function capturePhoto() {
		if (!videoEl || !stream || isCapturing) return;
		isCapturing = true;

		try {
			const videoWidth = videoEl.videoWidth;
			const videoHeight = videoEl.videoHeight;

			// Passport photo aspect ratio is 3:4 (e.g. 300x400)
			const targetHeight = videoHeight;
			const targetWidth = Math.round(videoHeight * 0.75); // 3:4
			const startX = Math.max(0, Math.round((videoWidth - targetWidth) / 2));

			// Create high-res in-memory canvas
			const canvas = document.createElement('canvas');
			canvas.width = targetWidth;
			canvas.height = targetHeight;
			const ctx = canvas.getContext('2d');

			// Fehlender 2D-Kontext war vorher der einzige Weg, der still endete: kein Foto,
			// keine Meldung, Knopf für immer grau. Lieber eine Fehlermeldung mit
			// „Erneut versuchen" als ein Dialog, in dem nichts mehr passiert.
			if (!ctx) {
				throw new Error('Zeichenfläche nicht verfügbar');
			}

			// Draw cropped area from video stream
			ctx.drawImage(videoEl, startX, 0, targetWidth, targetHeight, 0, 0, targetWidth, targetHeight);

			// Export high-quality JPEG
			const dataUrl = canvas.toDataURL('image/jpeg', 0.95);

			// Upload to backend
			const res = await apiClient.post(`/api/schueler/${studentId}/photo`, {
				photo_data: dataUrl
			});

			if (!res.ok) {
				throw new Error((await res.text()) || 'Upload fehlgeschlagen');
			}

			const data = await res.json();
			onCapture(data.url);
		} catch (err) {
			const error = /** @type {any} */ (err);
			errorMsg = 'Aufnahme fehlgeschlagen: ' + error.message;
		} finally {
			// Auch im Erfolgsfall zurücksetzen. Heute schließt der Aufrufer das Overlay
			// direkt danach, die Komponente verschwindet also ohnehin — aber daran darf
			// der Aufnahmeknopf nicht hängen.
			isCapturing = false;
		}
	}

	function stopCamera() {
		if (stream) {
			for (const track of stream.getTracks()) {
				track.stop();
			}
			stream = null;
		}
	}

	onMount(() => {
		startCamera();
		return () => stopCamera();
	});
</script>

<Modal open={true} onclose={onClose} size="lg" ebene="darueber" beschriftetDurch="webcam-titel">
	{#snippet header()}
		<h3 id="webcam-titel" class="text-base font-bold text-on-surface">Passbild aufnehmen</h3>
	{/snippet}
	<!-- Hinter der Sperre läuft keine Kamera (wie KameraScanner). -->
	<div class="space-y-4 p-6" use:ruhtWennTraege={{ anhalten: stopCamera, fortsetzen: startCamera }}>
		{#if errorMsg}
			<div
				role="alert"
				class="flex items-center gap-2 rounded-xl bg-error-container px-4 py-3 text-sm text-on-error-container"
			>
				<AlertTriangle class="h-4 w-4 shrink-0" aria-hidden="true" />
				<span>{errorMsg}</span>
			</div>
			<div class="flex justify-end">
				<Button variant="secondary" onclick={startCamera}>Erneut versuchen</Button>
			</div>
		{:else}
			<!-- Der Sucher mit den Hilfslinien für das Gesicht. -->
			<div class="relative aspect-video w-full overflow-hidden rounded-xl bg-black">
				<video bind:this={videoEl} autoplay playsinline class="h-full w-full object-cover"></video>
				<div
					class="pointer-events-none absolute inset-0 flex items-center justify-center bg-zinc-950/20"
				>
					<div
						class="relative flex h-[90%] w-[50%] items-center justify-center rounded-lg border-2 border-dashed border-emerald-400"
					>
						<div
							class="h-[80%] w-[85%] rounded-full border border-dashed border-emerald-400/40"
						></div>
						<span
							class="absolute bottom-2 rounded-full bg-zinc-950/90 px-2 py-0.5 text-[8px] font-bold tracking-wider text-emerald-400"
							>Gesichtsrahmen</span
						>
					</div>
				</div>
			</div>

			<div class="flex items-center justify-between gap-4">
				<p class="text-sm text-on-surface-variant">
					Das Foto wird auf das Hochformat 3:4 zugeschnitten.
				</p>
				<Button onclick={capturePhoto} disabled={isCapturing} class="shrink-0">
					{#if isCapturing}
						<Ladekreis size="sm" farbe="aktuell" />
						Speichert …
					{:else}
						<Camera class="h-4 w-4" aria-hidden="true" />
						Foto aufnehmen
					{/if}
				</Button>
			</div>
		{/if}
	</div>
</Modal>
