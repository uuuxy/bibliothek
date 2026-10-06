<script>
	/**
	 * Die Kamera an der Theke.
	 *
	 * Hier liegt der Rahmen, die Technik in `KameraScanner`: ein Erkenner für Theke und
	 * Inventur. Er fragt zuerst den eingebauten Erkenner des Browsers und fällt sonst auf ZXing
	 * zurück, über das ganze Bild und mit der Formatliste der Anwendung (barcode_detector.js),
	 * darunter Code 39, das Format der älteren Ausweise.
	 */
	import KameraScanner from '../inventur/lib/components/scanner/KameraScanner.svelte';
	import { omniboxStore } from './stores/omnibox.svelte.js';
	import { X } from '@lucide/svelte';
	import { onDestroy } from 'svelte';

	let { stopCamera, queryVal = $bindable(), submitAction } = $props();

	/** @type {any} */
	let scanner = $state(null);
	let meldung = $state('Kamera wird gestartet …');

	// Der Griff zum Abschalten liegt am Store, solange die Kamera offen ist: „Theke leeren"
	// und der Sperrbildschirm müssen den Strom abwürgen können, ohne dieses Bauteil zu
	// kennen. Ein Scanner, der hinter der Sperre weiterläuft, bucht ein vorgehaltenes Buch —
	// deshalb hängt hier ein Gate dran (idleLock.test.js).
	$effect(() => {
		if (scanner) omniboxStore.cameraScanner = { stop: () => scanner?.stopScanner() };
	});
	onDestroy(() => {
		omniboxStore.cameraScanner = null;
	});

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
	class="schema-dunkel w-full rounded-2xl overflow-hidden shadow-lg animate-slide-up bg-surface relative"
>
	<div class="absolute top-3 right-3 z-10">
		<button
			type="button"
			onclick={schliessen}
			class="icon-btn bg-secondary-container text-on-secondary-container"
			data-tip="Kamera schließen"
			aria-label="Kamera schließen"
		>
			<X class="h-5 w-5" aria-hidden="true" />
		</button>
	</div>
	<!-- Eine Zeile, die den Zustand sagt, statt einer festen Aufschrift: Eine Kamera, die
	     läuft und nichts findet, sähe sonst aus wie eine, die nicht gestartet ist. -->
	<div class="px-4 pt-3 pb-1 text-xs text-primary font-semibold text-center">
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
