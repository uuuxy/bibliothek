<script>
	/**
	 * @component KategorieRahmen
	 * Die Hülle EINER Einstellungs-Kategorie: Überschrift, ein Satz, der Inhalt und
	 * — falls die Kategorie etwas zu speichern hat — ihr eigener Speichern-Knopf.
	 *
	 * Warum je Kategorie gespeichert wird (Betreiber-Entscheidung 23.08.2026):
	 * Vorher lag EIN Knopf am Seitenende unter sieben fremden Abschnitten. Wer die
	 * Preiserfassung umlegte, speicherte Schuladresse, Fristen und Löschfristen mit.
	 * Und weil dabei immer alles auf einmal ging, trug das Formular drei verschiedene
	 * Leer-Regeln, die man nur aus dem Hinweistext erfuhr (Schule: leer = unverändert,
	 * Öffentliche Adresse: leer = abschalten, Datenschutz: 0 = aus, leer = unverändert).
	 * Jetzt schickt jede Kategorie ausschließlich ihre eigenen Felder — und damit gilt
	 * überall dieselbe Regel: Was im Feld steht, wird gespeichert.
	 *
	 * Ein Satz Erklärtext, nicht sechs Zeilen. Was darüber hinausgeht, steht im
	 * „Mehr"-Aufklapper — ein natives <details>, damit es ohne JavaScript, mit
	 * Tastatur und im Ausdruck funktioniert. Lange Erklärtexte sind ein Symptom:
	 * Sie beschreiben Verhalten, das die Struktur nicht von selbst zeigt.
	 *
	 * @prop {string} titel
	 * @prop {string} kurz - Supporting Text, EIN Satz.
	 * @prop {import('svelte').Snippet} [mehr] - Ausführliche Erläuterung im Aufklapper.
	 * @prop {() => Promise<void>} [speichern] - Ohne diese Funktion trägt die Kategorie
	 *   keinen Knopf (Prüf- und Werkzeugseiten wie die Betriebsbereitschaft).
	 * @prop {import('svelte').Snippet} children
	 */
	import Button from '../ui/Button.svelte';

	let { titel, kurz, mehr = undefined, speichern = undefined, children } = $props();

	let laeuft = $state(false);

	async function speichernKlick() {
		if (!speichern) return;
		laeuft = true;
		try {
			await speichern();
		} finally {
			laeuft = false;
		}
	}
</script>

<section class="flex w-full max-w-4xl flex-col gap-8">
	<header class="flex flex-col gap-1.5">
		<h2 class="text-lg font-medium text-on-surface">{titel}</h2>
		<p class="text-sm text-on-surface-variant">{kurz}</p>
		{#if mehr}
			<details class="group mt-1">
				<summary
					class="w-fit cursor-pointer list-none text-sm font-medium text-primary hover:underline"
				>
					<span class="group-open:hidden">Mehr</span>
					<span class="hidden group-open:inline">Weniger</span>
				</summary>
				<div class="mt-2 flex flex-col gap-2 text-sm text-on-surface-variant">
					{@render mehr()}
				</div>
			</details>
		{/if}
	</header>

	{@render children()}

	{#if speichern}
		<div class="flex justify-end border-t border-outline-variant pt-6">
			<Button onclick={speichernKlick} disabled={laeuft}>
				{laeuft ? 'Wird gespeichert …' : `${titel} speichern`}
			</Button>
		</div>
	{/if}
</section>
