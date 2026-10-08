<!-- @component KollegiumFormFelder — die Eingabefelder für jede Art im Kollegium.

     Bewusst kurz. Was hier NICHT steht, steht auch nicht zufällig woanders:

       - Keine Klasse und kein Abgangsjahr. Mit einer Klasse stünde die Lehrkraft in den
         Klassenlisten und im LUSD-Abgleich.
       - Kein Geburtsdatum. Es ist der Schlüssel des LUSD-Imports, und ein Kollege kommt
         nie aus der LUSD — es zu erheben, hätte keinen Zweck.
       - Keine Rolle. Eine Rolle vergibt der Administrator eigens; wer keine hat, ist
         Kollegium — der Grundzustand jeder Lehrkraft.

     Die SCHUL-E-MAIL steht seit dem 16.09.2026 hier, und sie ist Pflicht . Sie ist
     keine Kontaktangabe, sondern der Schlüssel: Aus ihr entsteht das Anmeldekonto, und
     weil sie eindeutig ist, findet die spätere Selbstanmeldung über „Mein Portal" genau
     diesen Eintrag wieder. Ohne sie stand die Person danach zweimal in der Leserdatei —
     Ausweis und Ausleihen am ersten Eintrag, die Anmeldung am zweiten, und niemand merkte
     es. Zwei Türen zu derselben Identität gibt es trotzdem nicht: Die Adresse wird an
     EINER Stelle geführt, nämlich am Konto.

     Praktikum und Fachbereich bekommen keinen Zugang (30.09.2026, artMitKonto): Das Feld ist
     bei ihnen verschlossen, und es entsteht nur der Eintrag in der Leserdatei. -->
<script>
	import Feld from '../ui/Feld.svelte';
	import { Info } from '@lucide/svelte';
	import { artMitKonto } from '../../leserArt.js';

	/** @type {{ art?: string, vorname: string, nachname: string, barcode: string, email: string }} */
	let {
		art = 'lehrkraft',
		vorname = $bindable(),
		nachname = $bindable(),
		barcode = $bindable(),
		email = $bindable()
	} = $props();

	const mitKonto = $derived(artMitKonto(art));
	// Ein Fachbereich ist keine Person: Die Beispiele zeigen, wie sein Name gemeint ist.
	const fachbereich = $derived(art === 'fachbereich');
</script>

<Feld
	label="Vorname *"
	bind:value={vorname}
	placeholder={fachbereich ? 'z.B. Fachbereich' : 'z.B. Katrin'}
/>

<Feld
	label="Nachname *"
	bind:value={nachname}
	placeholder={fachbereich ? 'z.B. Erdkunde' : 'z.B. Wendland'}
/>

<Feld
	label={mitKonto ? 'Schul-E-Mail *' : 'Schul-E-Mail'}
	type="email"
	bind:value={email}
	placeholder={mitKonto ? 'vorname.nachname@schule.de' : ''}
	disabled={!mitKonto}
	hint={mitKonto
		? 'Damit entsteht der Zugang zu „Mein Portal“ — und die Person steht später nicht doppelt da.'
		: 'Praktikum und Fachbereich bekommen keinen Zugang zu „Mein Portal“.'}
/>

<Feld
	label="Ausweisnummer (optional)"
	bind:value={barcode}
	placeholder="Wird automatisch vergeben, wenn leer"
/>

<div
	class="flex items-start gap-3 rounded-xl border border-outline-variant px-4 py-3 text-sm text-on-surface-variant"
>
	<Info class="h-5 w-5 shrink-0 text-outline" aria-hidden="true" />
	{#if mitKonto}
		<p>
			Es entsteht ein Eintrag in der Leserdatei — damit an der Theke Bücher auf diese Person gehen
			können — <span class="font-semibold">und ihr Zugang zu „Mein Portal“</span>. Anmelden wird sie
			sich mit ihrer Schuladresse und ihrem Mail-Passwort; ein Passwort speichern wir nicht. Eine
			<span class="font-semibold">Rolle</span> bekommt sie hier nicht — die vergibt der Administrator
			eigens in Benutzer &amp; Rechte.
		</p>
	{:else}
		<p>
			Es entsteht nur ein Eintrag in der Leserdatei, damit an der Theke Bücher darauf gebucht werden
			können. Einen Zugang zu „Mein Portal“ gibt es nicht.
		</p>
	{/if}
</div>
