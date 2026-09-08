<!-- @component Tabelle — DIE Datentabelle der Anwendung (Material 3 Data Table, dicht).

     Bis zum 08.09.2026 hatten 24 Tabellen 24 Rezepte: Das Aussehen hing mal am <thead>,
     mal an der Kopfzeile, mal an jedem <th>; sechs Innenabstände (px-4 py-2, p-4, p-4.5,
     py-3 px-4, px-3 py-2, py-2 pr-3), zwei Schriftgrößen, Zeilen-Hover in drei Farben
     (slate-50, blue-50, surface-container), gewählte Zeile in zwei. Nebeneinander auf
     einer Seite (Schülerdatei ↔ Mahnwesen ↔ Etiketten) las sich das als drei Programme.

     Ein Rezept, hier in <style> festgeschrieben, damit die Zellen NICHTS mehr über ihr
     Aussehen sagen müssen und es auch nicht können sollen (Ratsche
     frontend-hygiene-tabellen.test.js):
       - Kopfzelle: 40 px hoch, label 12 px medium in `on-surface-variant`, Trennlinie
         `outline-variant` (typo-rollen.spec.js: th 12, td 14).
       - Zelle: 14 px in `on-surface`, py-2 → Zeilen von 36–40 px, unsere Dichte.
       - Trennlinien zwischen den Zeilen in `outline-variant` (M3 Divider), keine
         Zebrastreifen, kein Rahmen um die Tabelle (edge-to-edge, siehe MahnwesenTable).
       - Hover: State-Layer 8 % `on-surface` — dieselbe Regel wie bei den Knöpfen.
       - Gewählt: `aria-selected="true"` an der Zeile → `secondary-container`, wie in
         M3-Listen und wie die 21 Stellen, die es schon so machten.
       - Waagerechter Innenabstand 16 px; die erste und letzte Zelle bekommen ihn auch,
         damit Text und Kopf am Seitenrand auf einer Linie stehen.
     Was die Zellen noch selbst bestimmen: Breite (w-*), Ausrichtung (text-right),
     Umbruch (whitespace-nowrap, truncate), Zahlen (tabular-nums, font-mono).

     `sticky` hält den Kopf beim Scrollen oben (Etikettenliste, Statistik).
     Der Aufrufer schreibt <thead>/<tbody>/<tr>/<th>/<td> wie gehabt — nur ohne Optik. -->
<script>
	/** @type {{ sticky?: boolean, class?: string, children: import('svelte').Snippet, [rest: string]: any }} */
	let { sticky = false, class: klasse = '', children, ...rest } = $props();
</script>

<table
	class="w-full border-collapse text-left text-sm text-on-surface {klasse}"
	class:sticky
	{...rest}
>
	{@render children()}
</table>

<style>
	table :global(th) {
		height: 2.5rem;
		padding: 0 1rem;
		font-size: var(--text-xs);
		line-height: var(--text-xs--line-height);
		font-weight: 500;
		color: var(--color-on-surface-variant);
		border-bottom: 1px solid var(--color-outline-variant);
		white-space: nowrap;
		vertical-align: middle;
	}
	table.sticky :global(th) {
		position: sticky;
		top: 0;
		z-index: 10;
		background-color: var(--color-surface);
	}
	table :global(td) {
		padding: 0.5rem 1rem;
		border-bottom: 1px solid var(--color-outline-variant);
		vertical-align: middle;
	}
	table :global(tbody tr) {
		transition: background-color 100ms var(--default-transition-timing-function);
	}
	table :global(tbody tr:hover) {
		background-color: color-mix(in oklab, var(--color-on-surface) 8%, transparent);
	}
	table :global(tbody tr[aria-selected='true']) {
		background-color: var(--color-secondary-container);
	}
	table :global(tbody tr[aria-selected='true']:hover) {
		background-color: color-mix(
			in oklab,
			var(--color-on-surface) 8%,
			var(--color-secondary-container)
		);
	}
</style>
