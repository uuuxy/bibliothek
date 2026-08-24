<script>
	/**
	 * @file PrintPreview.svelte
	 * Versteckte Druck-Ausgabe-Sektionen (Ausweis-Designer, Stapeldruck der Schülerdatei),
	 * die von der Druck-Engine gerendert werden. Der Auslöser setzt data-print-mode/-side
	 * am <body>; die CSS unten und styles/druck-ausweise.css zeigen die passende Sektion.
	 *
	 *   print-section-card      → Kartendrucker, Vorderseite
	 *   print-section-back-card → Kartendrucker, Rückseite
	 *
	 * Bis zum 24.08.2026 gab es zwei weitere Sektionen für einen A4-Bogen mit acht
	 * Kartenabbildern zum Ausschneiden. Er ist abgeschafft; an seiner Stelle steht der
	 * Etikettenbogen, und der entsteht serverseitig als PDF (schuelerEtiketten.js) und
	 * nicht über die Druck-CSS des Browsers. Deshalb bleiben hier nur die zwei Karten.
	 *
	 * Jede Sektion rendert pro Schüler eine Karte über CardFace — dieselbe Render-Quelle
	 * wie der profilseitige Einzeldruck (StudentPrintCard).
	 */
	import { idStore } from './idDesignerStore.svelte.js';
	import CardFace from './CardFace.svelte';

	/**
	 * `platzhalter` reicht der Ausweis-Designer durch: Sein Testdruck soll leere Bild- und
	 * Passbildfelder als Rahmen zeigen, der echte Stapeldruck aus der Schuelerdatei
	 * niemals (siehe CardFace.svelte).
	 *
	 * @type {{ students: any[], barcodeType: 'code39'|'qr', timestamp: number, platzhalter?: boolean }}
	 */
	const { students, barcodeType, timestamp, platzhalter = false } = $props();
</script>

<!-- Kartendrucker: Vorderseite -->
<div class="print-rendered-output print-section-card hidden print:block">
	{#each students as student (student.id)}
		<div class="print-card-box {idStore.front.theme}">
			<CardFace side="front" {student} {barcodeType} {timestamp} {platzhalter} />
		</div>
	{/each}
</div>

<!-- Kartendrucker: Rückseite (statische Elemente; kein personenbezogener Inhalt) -->
<div class="print-rendered-output print-section-back-card hidden">
	{#each students as _student (_student.id)}
		<div class="print-card-box {idStore.back.theme}">
			<CardFace side="back" student={null} {barcodeType} {timestamp} {platzhalter} />
		</div>
	{/each}
</div>

<style>
	@media print {
		:global(html, body) {
			margin: 0 !important;
			padding: 0 !important;
			background: white !important;
			overflow: hidden !important;
		}
		:global(main, .min-h-screen, .flex) {
			margin: 0 !important;
			padding: 0 !important;
			display: block !important;
			background: white !important;
			border: none !important;
			box-shadow: none !important;
		}
		:global(.no-print) {
			display: none !important;
		}
		:global(body[data-print-mode='card']) .print-section-card {
			display: block !important;
		}
		:global(body[data-print-mode='card'][data-print-side='back']) .print-section-card {
			display: none !important;
		}
		:global(body[data-print-mode='card'][data-print-side='back']) .print-section-back-card {
			display: block !important;
		}
	}
</style>
