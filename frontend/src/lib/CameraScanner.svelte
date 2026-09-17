<script>
	/**
	 * Die Kamera an der Theke.
	 *
	 * Bis zum 17.09.2026 hatte sie einen EIGENEN Erkenner (html5-qrcode mit einem
	 * Ausschnitt von 260×120 px), während der Inventur-Bereich einen zweiten benutzte.
	 * Zwei Scanner für dieselbe Aufgabe heißen zwei Fehlerbilder — und auf einem iPhone,
	 * das keinen eingebauten Barcode-Erkenner hat, blieb dieser hier stumm: kein Treffer,
	 * keine Meldung, nichts.
	 *
	 * Jetzt liegt hier der Rahmen und dort die Technik: `KameraScanner` fragt zuerst den
	 * eingebauten Erkenner des Browsers und fällt sonst auf ZXing zurück, über das GANZE
	 * Bild statt über einen schmalen Streifen, und mit der Formatliste der Anwendung
	 * (barcode_detector.js) — darin steht seit heute auch Code 39, das Format unserer
	 * eigenen Ausweise aus der Zeit vor Code 128.
	 */
	import KameraScanner from '../inventur/lib/components/scanner/KameraScanner.svelte';
	import { X } from '@lucide/svelte';

	let { stopCamera, queryVal = $bindable(), submitAction } = $props();

	/** @type {any} */
	let scanner = $state(null);
	let meldung = $state('Kamera wird gestartet …');

	function beiTreffer(code) {
		queryVal = String(code).trim();
		schliessen();
		submitAction();
	}

	async function schliessen() {
		try {
			await scanner?.stopScanner();
		} catch {
			// Ein Fehler beim Abschalten darf das Schliessen nicht aufhalten.
		}
		stopCamera();
	}
</script>

<div
	class="w-full rounded-2xl overflow-hidden border border-blue-200 shadow-lg animate-slide-up bg-black relative"
>
	<div class="absolute top-3 right-3 z-10">
		<button
			type="button"
			onclick={schliessen}
			class="p-1.5 rounded-full bg-white/80 text-slate-700 hover:bg-white shadow transition-colors cursor-pointer"
			title="Kamera schließen"
			aria-label="Kamera schließen"
		>
			<X class="h-5 w-5" aria-hidden="true" />
		</button>
	</div>
	<!-- EINE Zeile, die den Zustand sagt, statt einer festen Aufschrift: Eine Kamera, die
	     läuft und nichts findet, sieht sonst genauso aus wie eine, die gar nicht erst
	     gestartet ist — und genau das war am 17.09.2026 die Beschwerde („Kamera geht auf,
	     nichts passiert"). -->
	<div class="px-4 pt-3 pb-1 text-xs text-blue-200 font-semibold text-center">
		{meldung}
	</div>
	<div class="px-4 pb-4">
		<KameraScanner
			bind:this={scanner}
			onDecode={beiTreffer}
			onStatusChange={(text) => (meldung = text)}
			showControls={false}
		/>
	</div>
</div>
