<!-- @component KollegiumFormFelder — die Eingabefelder für eine Lehrkraft oder LiV.

     Bewusst kurz. Was hier NICHT steht, steht auch nicht zufällig woanders:

       - Keine Klasse und kein Abgangsjahr. Mit einer Klasse stünde die Lehrkraft in den
         Klassenlisten und im LUSD-Abgleich.
       - Kein Geburtsdatum. Es ist der Schlüssel des LUSD-Imports, und ein Kollege kommt
         nie aus der LUSD — es zu erheben, hätte keinen Zweck.
       - Keine E-Mail-Adresse und keine Rolle. An der E-Mail hängt die ANMELDUNG; sie
         gehört in „Benutzer & Rechte", damit es nicht zwei Türen zu derselben Identität
         gibt. Hier entsteht eine Leserzeile und kein Konto. -->
<script>
	import Feld from '../ui/Feld.svelte';
	import { Info } from '@lucide/svelte';

	/** @type {{ vorname: string, nachname: string, barcode: string }} */
	let { vorname = $bindable(), nachname = $bindable(), barcode = $bindable() } = $props();
</script>

<Feld label="Vorname *" bind:value={vorname} placeholder="z.B. Katrin" />

<Feld label="Nachname *" bind:value={nachname} placeholder="z.B. Wendland" />

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
		Hier entsteht ein Eintrag in der Leserdatei, damit an der Theke Bücher auf diese Person gehen
		können — <span class="font-semibold">kein Zugang zum Programm</span>. Den holt sie sich selbst
		über „Mein Portal“ mit ihrer Schuladresse; freigeschaltet wird er in Benutzer &amp; Rechte.
	</p>
</div>
