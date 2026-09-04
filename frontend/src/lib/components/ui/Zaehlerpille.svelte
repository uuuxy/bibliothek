<script>
	/**
	 * @component Zaehlerpille — das Zähler-Badge nach Material 3 („large badge").
	 *
	 * Es gab davon ZWEI Fassungen, und sie sind auseinandergelaufen: Die Seitenleiste
	 * trug 28 × 20 px auf `error`, die Reiterleiste 32 × 16 px auf `primary` — und nur
	 * die Seitenleiste kappte die Zahl. Im Druck-Center stand deshalb am 04.09.2026
	 * links „999+" und zwei Zentimeter daneben „30674": dieselbe Zahl, zwei Aussagen.
	 *
	 * Die Kappung ist keine Platzfrage, sondern eine Aussage — der Satz stand schon in
	 * der Seitenleiste und gilt hier weiter: Ob 30.674 oder 12.000 Etiketten offen sind,
	 * ändert keine Entscheidung. „Mehr als du heute schaffst" ist die ganze Information.
	 * Eine fünfstellige Zahl macht aus der 16-px-Pille einen 45 px langen Strich.
	 *
	 * Die volle Zahl geht nicht verloren: Sie steht im aria-label, damit ein Screenreader
	 * sie vorliest.
	 *
	 * Maße nach M3: 16 dp Höhe, 4 dp Polsterung, label-small, voll gerundet, und die
	 * Farbrolle ist `error` — auch am Reiter. M3 kennt für das Badge keine zweite.
	 *
	 * @prop {number} anzahl - 0 oder weniger zeigt nichts (M3: kein leeres Badge).
	 * @prop {string} [beschreibung] - was gezählt wird, z. B. "Etiketten offen".
	 * @prop {string} [klasse]
	 */
	/** @type {{ anzahl: number, beschreibung?: string, klasse?: string }} */
	let { anzahl, beschreibung = 'offen', klasse = '' } = $props();

	const KAPPUNG = 999;
	const text = $derived(anzahl > KAPPUNG ? `${KAPPUNG}+` : String(anzahl));
</script>

{#if anzahl > 0}
	<span
		class="inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-error px-1 text-label-small font-medium text-on-error {klasse}"
		aria-label="{anzahl} {beschreibung}"
	>
		{text}
	</span>
{/if}
