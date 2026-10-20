<!-- @component Tabelle — die Datentabelle der Anwendung (Material 3 Data Table, dicht).

     Ein Rezept, hier in <style> festgeschrieben, damit die Zellen nichts über ihr Aussehen
     sagen müssen und es auch nicht können sollen (Ratsche frontend-hygiene-tabellen.test.js):
     Tabellen mit eigenen Abständen und Farben lesen sich nebeneinander als drei Programme.
       - Kopfzelle: 40 px hoch, label 12 px medium in `on-surface-variant`, Trennlinie
         `outline-variant` (typo-rollen.spec.js: th 12, td 14).
       - Zelle: 14 px in `on-surface`, py-2 → Zeilen von 36–40 px, unsere Dichte.
       - Trennlinien zwischen den Zeilen in `outline-variant` (M3 Divider), keine
         Zebrastreifen, kein Rahmen um die Tabelle (edge-to-edge, siehe MahnwesenTabelle).
       - Hover: State-Layer 8 % `on-surface` — dieselbe Regel wie bei den Knöpfen.
       - Gewählt: `aria-selected="true"` an der Zeile → `secondary-container`, wie in
         M3-Listen.
       - Waagerechter Innenabstand 16 px; die erste und letzte Zelle bekommen ihn auch,
         damit Text und Kopf am Seitenrand auf einer Linie stehen.
     Was die Zellen noch selbst bestimmen: Breite (w-*), Ausrichtung (text-right; senkrecht
     align-top an Zeile oder Zelle, sonst mittig), Umbruch (whitespace-nowrap, truncate),
     Zahlen (tabular-nums, font-mono).

     `sticky` hält den Kopf beim Scrollen oben (Etikettenliste, Statistik).
     Der Aufrufer schreibt <thead>/<tbody>/<tr>/<th>/<td> wie gehabt — nur ohne Optik.

     Barrierefreiheit (Gate e2e/barrierefreiheit-dialog.spec.js):
       - `beschriftung` ist Pflicht (Ratsche frontend-hygiene-tabellen.test.js) und wird
         als unsichtbare <caption> gesetzt: Ein Screenreader nennt beim Betreten der
         Tabelle, was hier steht; sehende Nutzer brauchen den Titel nicht, die Seite sagt es.
       - Jede <th> bekommt ihren `scope` (col im Kopf, row im Rumpf) von der Aktion
         unten, damit 147 Kopfzellen ihn nicht einzeln tragen müssen; ein selbst
         gesetzter scope bleibt stehen. Ein MutationObserver hält das auch für Zeilen,
         die später kommen. -->
<script>
	/** @type {{ beschriftung: string, sticky?: boolean, class?: string, children: import('svelte').Snippet, [rest: string]: any }} */
	let { beschriftung, sticky = false, class: klasse = '', children, ...rest } = $props();

	/**
	 * scope="col" für Kopfzellen im <thead>, scope="row" für <th> im Rumpf.
	 * @param {HTMLTableElement} tabelle
	 */
	function kopfzellenScope(tabelle) {
		const setze = () => {
			for (const th of tabelle.querySelectorAll('th')) {
				if (!th.hasAttribute('scope'))
					th.setAttribute('scope', th.closest('thead') ? 'col' : 'row');
			}
		};
		setze();
		const beobachter = new MutationObserver(setze);
		beobachter.observe(tabelle, { childList: true, subtree: true });
		return { destroy: () => beobachter.disconnect() };
	}
</script>

<table
	class="w-full border-collapse text-left text-sm text-on-surface {klasse}"
	class:sticky
	use:kopfzellenScope
	{...rest}
>
	<caption class="sr-only">{beschriftung}</caption>
	{@render children()}
</table>

<style>
	/* Bezugsrahmen für die unsichtbare Beschriftung: Sie ist absolut gesetzt, hinge sonst an
	   einem fernen Vorfahren und verlängerte die Seite, sobald die Tabelle unter dem
	   Fensterrand liegt. */
	table {
		position: relative;
	}
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
	/* Ohne vertical-align: Mittig ist der Browser-Standard, und eine Regel hier schlüge jedes
	   align-top des Aufrufers, weil Bauteil-Stile außerhalb der Tailwind-Schichten stehen. */
	table :global(td) {
		padding: 0.5rem 1rem;
		border-bottom: 1px solid var(--color-outline-variant);
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
