<!-- @component SignaturFeld — die Signatur steht physisch auf dem Buchrücken-Etikett
     und ist bei Neuanlage eines Bibliotheksbuchs Pflicht. Lernmittel tragen kein
     Etikett (Migration 093): für sie ist das Feld frei.

     Die Vorschläge sind die im BESTAND vorkommenden Signaturen (GET /api/signaturen) —
     die Littera-Codes am Regal („Sk", „JF", „MANGA"). Bis zum 22.09.2026 schlug das
     Programm stattdessen „BIB {Kategorie}" vor, ein Wort, das in keinem Regal steht;
     damit entstand neben Litteras Vokabular ein zweites. Eine neue Adresse lässt sich
     weiterhin frei eintippen.

     Der Rahmen wechselt die Farbe statt nur eine Meldung darunter zu setzen: Ohne
     Signatur ist Speichern gesperrt, das muss man sehen, bevor man klickt. -->
<script>
	import { Tag } from '@lucide/svelte';
	import Feld from '../../../../lib/components/ui/Feld.svelte';
	import { ladeSignaturen } from '../../../../lib/utils/signaturen.js';

	/** @type {{ formular: any, signaturFehlt: boolean }} */
	let { formular = $bindable(), signaturFehlt } = $props();

	/** @type {{ signatur: string, titel: number }[]} */
	let vorhandene = $state([]);
	$effect(() => {
		let abgebrochen = false;
		ladeSignaturen().then((liste) => {
			if (!abgebrochen) vorhandene = liste;
		});
		return () => {
			abgebrochen = true;
		};
	});
	// Platzhalter aus dem Bestand: die drei größten Regale. Ohne Bestand kein Beispiel —
	// ein erfundenes wäre genau der Fehler, den diese Änderung abschafft.
	const beispiele = $derived(
		[...vorhandene]
			.sort((a, b) => b.titel - a.titel)
			.slice(0, 3)
			.map((s) => s.signatur)
			.join(', ')
	);
</script>

<div
	class="rounded-xl border-2 p-4 transition-colors {signaturFehlt
		? 'border-rose-300 bg-rose-50/40'
		: 'border-emerald-200 bg-emerald-50/30'}"
>
	<label for="buch-signatur" class="flex items-center gap-2 text-sm font-bold text-slate-800 mb-1">
		<Tag class="h-4 w-4 text-slate-500" aria-hidden="true" />
		Signatur (Buchrücken)
		{#if !formular.id && !formular.istLernmittel}<span
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
		list="signatur-vorschlaege"
		placeholder={beispiele ? `z. B. ${beispiele}` : 'Signatur des Regals'}
		ungueltig={signaturFehlt}
		hint={signaturFehlt
			? 'Ohne Signatur kein Etikett — bitte Systematik-Kürzel eintragen (Speichern ist bis dahin gesperrt).'
			: formular.istLernmittel
				? 'Lernmittel tragen kein Rückenetikett — die Signatur ist hier nur eine Notiz.'
				: 'Wird 1:1 auf das Rücken-Etikett gedruckt — am besten eine vorhandene Regaladresse.'}
	/>
	<datalist id="signatur-vorschlaege">
		{#each vorhandene as s (s.signatur)}
			<option value={s.signatur}>{s.signatur} — {s.titel} Titel</option>
		{/each}
	</datalist>
</div>
