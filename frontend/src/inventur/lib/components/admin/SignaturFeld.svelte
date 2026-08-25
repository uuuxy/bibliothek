<!-- @component SignaturFeld — die Signatur steht physisch auf dem Buchrücken-Etikett
     und ist bei Neuanlage Pflicht. Die DNB-Altersstufe füllt höchstens einen
     „BIB …"-Vorschlag vor (IsbnFeld); hier entscheidet sich, ob das Buch zur
     Littera-Systematik passt.

     Der Rahmen wechselt die Farbe statt nur eine Meldung darunter zu setzen: Ohne
     Signatur ist Speichern gesperrt, das muss man sehen, bevor man klickt. -->
<script>
	import { Tag } from '@lucide/svelte';
	import Feld from '../../../../lib/components/ui/Feld.svelte';

	/** @type {{ formular: any, signaturFehlt: boolean, autorKuerzel: string }} */
	let { formular = $bindable(), signaturFehlt, autorKuerzel } = $props();
</script>

<div
	class="rounded-xl border-2 p-4 transition-colors {signaturFehlt
		? 'border-rose-300 bg-rose-50/40'
		: 'border-emerald-200 bg-emerald-50/30'}"
>
	<label for="buch-signatur" class="flex items-center gap-2 text-sm font-bold text-slate-800 mb-1">
		<Tag class="h-4 w-4 text-slate-500" aria-hidden="true" />
		Signatur (Buchrücken)
		{#if !formular.id}<span
				class="text-xs font-medium px-1.5 py-0.5 rounded {signaturFehlt
					? 'bg-rose-100 text-rose-700'
					: 'bg-emerald-100 text-emerald-700'}">Pflicht</span
			>{/if}
	</label>
	<!-- Beschriftung bleibt eigenes <label>: Symbol und „Pflicht"-Marke passen nicht in
	     die Text-Prop des Feldes. Fehlerzustand und Hinweis kommen aus dem Bauteil. -->
	<Feld
		id="buch-signatur"
		bind:value={formular.signatur}
		placeholder={autorKuerzel
			? `z. B. "${autorKuerzel}" (Belletristik) oder "LMF M"`
			: 'z. B. LMF M, BIB ROM, Row …'}
		ungueltig={signaturFehlt}
		hint={signaturFehlt
			? 'Ohne Signatur kein Etikett — bitte Systematik-Kürzel eintragen (Speichern ist bis dahin gesperrt).'
			: 'Wird 1:1 auf das Rücken-Etikett gedruckt. Bestehende Littera-Signaturen werden von Importen nie überschrieben.'}
	/>
</div>
