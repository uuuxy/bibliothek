<!-- @component KollegiumFormFelder — die Eingabefelder für eine Lehrkraft oder LiV.

     Bewusst kurz. Was hier NICHT steht, steht auch nicht zufällig woanders:

       - Keine Klasse und kein Abgangsjahr. Mit einer Klasse stünde die Lehrkraft in den
         Klassenlisten und im LUSD-Abgleich.
       - Kein Geburtsdatum. Es ist der Schlüssel des LUSD-Imports, und ein Kollege kommt
         nie aus der LUSD — es zu erheben, hätte keinen Zweck.
       - Keine Rolle. Eine Rolle vergibt der Administrator eigens; wer keine hat, ist
         Kollegium — der Grundzustand jeder Lehrkraft.

     Die SCHUL-E-MAIL steht seit dem 16.09.2026 hier, und sie ist Pflicht (Peter). Sie ist
     keine Kontaktangabe, sondern der Schlüssel: Aus ihr entsteht das Anmeldekonto, und
     weil sie eindeutig ist, findet die spätere Selbstanmeldung über „Mein Portal" genau
     diesen Eintrag wieder. Ohne sie stand die Person danach zweimal in der Leserdatei —
     Ausweis und Ausleihen am ersten Eintrag, die Anmeldung am zweiten, und niemand merkte
     es. Zwei Türen zu derselben Identität gibt es trotzdem nicht: Die Adresse wird an
     EINER Stelle geführt, nämlich am Konto. -->
<script>
	import Feld from '../ui/Feld.svelte';
	import { Info } from '@lucide/svelte';

	/** @type {{ vorname: string, nachname: string, barcode: string, email: string }} */
	let {
		vorname = $bindable(),
		nachname = $bindable(),
		barcode = $bindable(),
		email = $bindable()
	} = $props();
</script>

<Feld label="Vorname *" bind:value={vorname} placeholder="z.B. Katrin" />

<Feld label="Nachname *" bind:value={nachname} placeholder="z.B. Wendland" />

<Feld
	label="Schul-E-Mail *"
	type="email"
	bind:value={email}
	placeholder="vorname.nachname@schule.de"
	hint="Damit entsteht der Zugang zu „Mein Portal“ — und die Person steht später nicht doppelt da."
/>

<Feld
	label="Ausweisnummer (optional)"
	bind:value={barcode}
	placeholder="Wird automatisch vergeben, wenn leer"
/>

<div
	class="flex items-start gap-3 rounded-xl border border-outline-variant bg-surface-container-low px-4 py-3 text-sm text-on-surface-variant"
>
	<Info class="h-5 w-5 shrink-0 text-outline" aria-hidden="true" />
	<p>
		Es entsteht ein Eintrag in der Leserdatei — damit an der Theke Bücher auf diese Person gehen
		können — <span class="font-semibold">und ihr Zugang zu „Mein Portal“</span>. Anmelden wird sie
		sich mit ihrer Schuladresse und ihrem Mail-Passwort; ein Passwort speichern wir nicht. Eine
		<span class="font-semibold">Rolle</span> bekommt sie hier nicht — die vergibt der Administrator eigens
		in Benutzer &amp; Rechte.
	</p>
</div>
