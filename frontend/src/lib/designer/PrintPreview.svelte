<script>
	/**
	 * @file PrintPreview.svelte
	 * Versteckte Druck-Ausgabe-Sektionen (Batch, DruckCenter), die von der Druck-Engine
	 * gerendert werden. triggerPrint() setzt data-print-mode/-side am <body>; die CSS
	 * unten (plus app.css) zeigt selektiv die passende Sektion.
	 *
	 *   print-section-card      → Kartendrucker, Vorderseite
	 *   print-section-a4        → A4-Bogen,      Vorderseite
	 *   print-section-back-card → Kartendrucker, Rückseite
	 *   print-section-back-a4   → A4-Bogen,      Rückseite
	 *
	 * Jede Sektion rendert pro Schüler eine Karte über CardFace — dieselbe Render-Quelle
	 * wie der profilseitige Einzeldruck (StudentPrintCard).
	 */
	import { idStore } from './idDesignerStore.svelte.js';
	import CardFace from './CardFace.svelte';

	/** @type {{ students: any[], barcodeType: 'code39'|'qr', timestamp: number }} */
	const { students, barcodeType, timestamp } = $props();
</script>

<!-- Kartendrucker: Vorderseite -->
<div class="print-rendered-output print-section-card hidden print:block">
	{#each students as student (student.id)}
		<div class="print-card-box {idStore.front.theme}">
			<CardFace side="front" {student} {barcodeType} {timestamp} />
		</div>
	{/each}
</div>

<!-- A4-Bogen: Vorderseite -->
<div class="print-rendered-output print-section-a4 a4_sheet hidden print:block">
	<div class="print-cards-grid">
		{#each students as student (student.id)}
			<div class="print-card-box {idStore.front.theme}">
				<CardFace side="front" {student} {barcodeType} {timestamp} />
			</div>
		{/each}
	</div>
</div>

<!-- Kartendrucker: Rückseite (statische Elemente; kein personenbezogener Inhalt) -->
<div class="print-rendered-output print-section-back-card hidden">
	{#each students as _student (_student.id)}
		<div class="print-card-box {idStore.back.theme}">
			<CardFace side="back" student={null} {barcodeType} {timestamp} />
		</div>
	{/each}
</div>

<!-- A4-Bogen: Rückseite -->
<div class="print-rendered-output print-section-back-a4 a4_sheet hidden">
	<div class="print-cards-grid">
		{#each students as _student (_student.id)}
			<div class="print-card-box {idStore.back.theme}">
				<CardFace side="back" student={null} {barcodeType} {timestamp} />
			</div>
		{/each}
	</div>
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
		:global(body[data-print-mode='a4'][data-print-side='back']) .print-section-a4 {
			display: none !important;
		}
		:global(body[data-print-mode='card'][data-print-side='back']) .print-section-back-card {
			display: block !important;
		}
		:global(body[data-print-mode='a4'][data-print-side='back']) .print-section-back-a4 {
			display: block !important;
		}
	}
</style>
