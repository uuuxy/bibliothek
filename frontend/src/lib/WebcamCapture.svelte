<!-- @component WebcamCapture — Aufnahme des Passbilds mit der Webcam.

     Ein Dialog des Hauses (Modal). Dunkel ist nur der Sucher: Ein Kamerabild steht auf Schwarz,
     die Hilfslinien gehören zum Bild. Der Sucher zeigt das ganze Kamerabild; hell bleibt der
     Ausschnitt, der gespeichert wird (passbildAusschnitt.js). Der erste Fokus liegt auf
     „Schließen" im Kopf und nicht auf der Aufnahme: An der Theke tippt der Handscanner blind und
     endet mit Enter, das nähme sonst ein Foto auf und ersetzte das vorhandene. -->
<script>
	import { TriangleAlert, Camera } from '@lucide/svelte';
	import { apiClient } from './apiFetch.js';
	import Modal from './Modal.svelte';
	import Button from './components/ui/Button.svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import { onMount } from 'svelte';
	import { ruhtWennTraege } from './actions/ruhtWennTraege.js';
	import { passbildAusschnitt, PASSBILD_FORM } from './passbildAusschnitt.js';

	/** @type {{ studentId: string, onCapture: (url: string) => void, onClose: () => void }} */
	let { studentId, onCapture, onClose } = $props();

	/** @type {HTMLVideoElement | null} */
	let videoEl = $state(null);
	/** @type {MediaStream | null} */
	let stream = $state(null);
	/** @type {string | null} */
	let errorMsg = $state(null);
	let isCapturing = $state(false);
	// Breite zu Höhe des Kamerabilds. Der Sucher nimmt diese Form an, damit er das ganze Bild
	// zeigt; bis die Kamera sie nennt, gilt das angefragte Breitbild.
	let bildform = $state(16 / 9);
	const hochkant = $derived(bildform < PASSBILD_FORM);

	function formLesen() {
		if (videoEl?.videoWidth && videoEl.videoHeight) {
			bildform = videoEl.videoWidth / videoEl.videoHeight;
		}
	}

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
			const a = passbildAusschnitt(videoEl.videoWidth, videoEl.videoHeight);
			const canvas = document.createElement('canvas');
			canvas.width = a.breite;
			canvas.height = a.hoehe;
			const ctx = canvas.getContext('2d');

			// Fehlender 2D-Kontext war vorher der einzige Weg, der still endete: kein Foto,
			// keine Meldung, Knopf für immer grau. Lieber eine Fehlermeldung mit
			// „Erneut versuchen" als ein Dialog, in dem nichts mehr passiert.
			if (!ctx) {
				throw new Error('Zeichenfläche nicht verfügbar');
			}

			ctx.drawImage(videoEl, a.x, a.y, a.breite, a.hoehe, 0, 0, a.breite, a.hoehe);

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
				<TriangleAlert class="h-4 w-4 shrink-0" aria-hidden="true" />
				<span>{errorMsg}</span>
			</div>
			<div class="flex justify-end">
				<Button variant="secondary" onclick={startCamera}>Erneut versuchen</Button>
			</div>
		{:else}
			<!-- Der Sucher hat die Form des Kamerabilds und höchstens 60 % der Fensterhöhe. Über dem
			     Bild liegt links und rechts (hochkant: oben und unten) ein Schleier; der helle Bereich
			     dazwischen hat die Form des Passbilds, das Oval darin ist die Hilfslinie fürs Gesicht. -->
			<div
				class="relative mx-auto overflow-hidden rounded-xl bg-inverse-surface"
				style:aspect-ratio={bildform}
				style:width="min(100%, calc(60vh * {bildform}))"
			>
				<video
					bind:this={videoEl}
					autoplay
					playsinline
					onloadedmetadata={formLesen}
					onresize={formLesen}
					class="h-full w-full object-cover"
				></video>
				<div class="pointer-events-none absolute inset-0 flex {hochkant ? 'flex-col' : ''}">
					<div class="flex-1 bg-scrim/32"></div>
					<div
						data-testid="passbild-ausschnitt"
						class="flex aspect-3/4 shrink-0 items-center justify-center border border-outline-variant {hochkant
							? 'w-full'
							: 'h-full'}"
					>
						<!-- Helle Striche auf einem dunklen Ring: zu sehen vor heller wie vor dunkler Wand. -->
						<div
							class="h-[72%] w-[72%] rounded-[50%] border border-dashed border-outline-variant outline outline-scrim/32"
						></div>
					</div>
					<div class="flex-1 bg-scrim/32"></div>
				</div>
			</div>

			<div class="flex items-center justify-between gap-4">
				<p class="text-sm text-on-surface-variant">
					Gespeichert wird der helle Ausschnitt im Hochformat 3:4.
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
