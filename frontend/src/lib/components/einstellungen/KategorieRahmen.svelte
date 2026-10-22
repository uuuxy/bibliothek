<script>
	/**
	 * @component KategorieRahmen
	 * Die Hülle einer Einstellungs-Kategorie: Überschrift, ein Satz, der Inhalt und, falls
	 * die Kategorie etwas zu speichern hat, ihr eigener Speichern-Knopf.
	 *
	 * Warum je Kategorie gespeichert wird: Ein Knopf für alles speicherte Schuladresse,
	 * Fristen und Löschfristen mit, wenn jemand nur die Preiserfassung umlegt, und jede
	 * Sektion bräuchte eine eigene Leer-Regel als Notbremse. So schickt jede Kategorie
	 * ausschließlich ihre eigenen Felder, und überall gilt dieselbe Regel: Was im Feld
	 * steht, wird gespeichert.
	 *
	 * Ein Satz Erklärtext, nicht sechs Zeilen. Was darüber hinausgeht, steht im
	 * „Mehr"-Aufklapper — ein natives <details>, damit es ohne JavaScript, mit
	 * Tastatur und im Ausdruck funktioniert. Lange Erklärtexte sind ein Symptom:
	 * Sie beschreiben Verhalten, das die Struktur nicht von selbst zeigt.
	 *
	 * @prop {string} titel
	 * @prop {string} kurz - Supporting Text, EIN Satz.
	 * @prop {import('svelte').Snippet} [mehr] - Ausführliche Erläuterung im Aufklapper.
	 * @prop {() => import('../../einstellungenSpeichern.js').KategorieEingabe} [eingabe] - Die Felder
	 *   der Kategorie mit dem Stand beim Öffnen. Ohne sie trägt die Kategorie keinen Knopf
	 *   (Prüf- und Werkzeugseiten wie die Betriebsbereitschaft).
	 * @prop {() => void | Promise<void>} [onSaved] - Neu laden nach dem Speichern.
	 * @prop {import('svelte').Snippet} children
	 */
	import Button from '../ui/Button.svelte';
	import { speichereKategorie, hatAenderungen } from '../../einstellungenSpeichern.js';
	import { schuetzeFormulare } from '../../stores/verlassenSchutz.svelte.js';

	let {
		titel,
		kurz,
		mehr = undefined,
		eingabe = undefined,
		onSaved = undefined,
		children
	} = $props();

	// Ein Wechsel der Kategorie oder der Seite baut die Kategorie ab. Hält sie oder ein Formular
	// darin (meldeFormular) Ungespeichertes, fragt der Verlassen-Schutz vorher.
	schuetzeFormulare(() => (eingabe ? hatAenderungen(eingabe()) : false));

	let laeuft = $state(false);

	async function speichernKlick() {
		if (!eingabe) return;
		laeuft = true;
		try {
			await speichereKategorie({ ...eingabe(), onSaved });
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

	{#if eingabe}
		<div class="flex justify-end border-t border-outline-variant pt-6">
			<Button onclick={speichernKlick} disabled={laeuft}>
				{laeuft ? 'Wird gespeichert …' : `${titel} speichern`}
			</Button>
		</div>
	{/if}
</section>
