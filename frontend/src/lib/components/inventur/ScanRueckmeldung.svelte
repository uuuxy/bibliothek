<!--
  @component
  ScanRueckmeldung — was der letzte Scan der Inventur ergeben hat: in Ordnung, mit Hinweis
  oder abgelehnt.

  Drei Zustände, drei Rollenvierer aus styles/rollen.css: success, warning, error. M3,
  „Define custom color roles": „a static green color called Success is defined in addition to
  the scheme, and applied to UI to indicate a success state" — der Container als Fläche der
  Karte, die On-Farbe für ihren Inhalt. Bis zum 24.09.2026 stand die Karte in Palettenfarben
  (emerald, red, amber) in UnifiedInventory.svelte, die Weiche viermal ausgeschrieben.

  Eigenes Bauteil, weil keins in ui/ diese Aufgabe hat: StatusChip ist ein Chip, LadeFehler
  ein Ladezustand mit „Erneut", Snackbar eine flüchtige Meldung. Diese Karte bleibt stehen,
  bis der nächste Scan kommt.
-->
<script>
	import { Check, TriangleAlert, X } from '@lucide/svelte';

	/** @type {{ scan: { success: boolean, warnings: string[], title: string, barcode: string } }} */
	let { scan } = $props();

	// Ganze Klassennamen, nicht zusammengesetzt: Tailwind findet nur, was wörtlich im
	// Quelltext steht.
	const TOENE = {
		ok: {
			flaeche: 'bg-success-container text-on-success-container',
			marke: 'bg-success text-on-success',
			symbol: Check
		},
		hinweis: {
			flaeche: 'bg-warning-container text-on-warning-container',
			marke: 'bg-warning text-on-warning',
			symbol: TriangleAlert
		},
		abgelehnt: {
			flaeche: 'bg-error-container text-on-error-container',
			marke: 'bg-error text-on-error',
			symbol: X
		}
	};

	const ton = $derived(
		!scan.success ? TOENE.abgelehnt : scan.warnings.length > 0 ? TOENE.hinweis : TOENE.ok
	);
	const Symbol = $derived(ton.symbol);
</script>

<div class="rounded-2xl p-6 {ton.flaeche}">
	<div class="flex items-start space-x-4">
		<div class="p-2 rounded-full shrink-0 {ton.marke}">
			<Symbol class="w-6 h-6" aria-hidden="true" />
		</div>

		<div class="flex-1">
			<h4 class="text-lg font-bold">{scan.title}</h4>
			<p class="text-sm font-medium mt-1">Barcode: {scan.barcode}</p>

			{#if scan.warnings.length > 0}
				<ul class="mt-3 space-y-1">
					{#each scan.warnings as warn, i (i)}
						<li class="flex items-start text-sm">
							<span class="mr-2 mt-0.5">•</span>
							<span>{warn}</span>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	</div>
</div>
