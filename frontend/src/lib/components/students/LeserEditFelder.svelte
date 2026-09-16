<!-- @component LeserEditFelder — die Eingabefelder der Leserakte. EINE Maske für jeden.

     Bis zum 16.09.2026 war dieses Formular ein Schüler-Formular: Klasse, Abgangsjahr,
     LUSD-ID und Eltern-E-Mail standen fest darin. Ein Kollege kam gar nicht erst hierher
     (seine Akte hatte keinen Bearbeiten-Knopf) — und hätte er es, wäre er an der leeren
     Klasse gescheitert, die der Server als Pflichtfeld abweist. Absprache vom 16.09.2026:
     „die Leute die bereits eine Rolle haben stehen zwar in der ausleihliste, ich kann
     dort aber keine adressedaten etc nachtragen."

     Mein erster Versuch blendete die Felder je nach Art ein und aus — zwei Masken. Absprache:
     „warum eine andere maske als bei schülern? das ist doch schon wieder viel zu
     kompliziert." Er hat recht: Zwingend ist nur EIN Unterschied, der Rest war Vorsicht.

     Deshalb steht hier für jeden dasselbe an derselben Stelle. Abgeschaltet statt
     versteckt ist nur, was wirklich nicht gehen darf:

       - LUSD-ID: Die Datenbank verbietet sie einem Kollegen
         (chk_leser_nur_schueler_werden_abgaenger). Durch die LUSD kommen nur Schüler.
       - Klasse und Abgangsjahr: „eine klasse muss ja keinem lehrer/liv zugeordnet
         werden" . Das Abgangsjahr hängt an der Klasse — der Server leitet es aus
         ihr ab (calculateAbgaengerJahr), also gehören beide zusammen.
       - Die Art über die Schüler-Grenze: Ein Schüler bleibt Schüler.

     REIHENFOLGE (16.09.2026, nach der Blick auf die fertige Maske): Die Art steht
     ZUERST, nicht am Ende der persönlichen Daten. An ihr hängt, welche Felder überhaupt
     leben — vorher las man erst die graue LUSD-ID mit dem Hinweis „Durch die LUSD kommen
     nur Schüler" und fand den Grund dafür eine Zeile darunter. Der Anlege-Dialog fragt
     seit demselben Tag ebenfalls zuerst nach der Art (StudentCreateModal.svelte), und
     das Bearbeiten ist die Stelle, an der die Art tatsächlich Felder sperrt.

     Die LUSD-ID ist dabei von den persönlichen Daten zu den SCHULDATEN gewandert: Sie ist
     keine Angabe über die Person, sondern die Kennung der Schulverwaltung — und sie stand
     auf dem prominentesten Platz der Seite, obwohl sie beim Kollegen immer tot ist. Die
     Ausweisnummer steht jetzt als LETZTES im Abschnitt: Bei einem Kollegen ist sie das
     einzige bediente Feld und stand vorher zwischen zwei gesperrten. -->
<script>
	import Feld from '../ui/Feld.svelte';
	import Abschnitt from '../ui/Abschnitt.svelte';
	import LeserArtWahl from './LeserArtWahl.svelte';
	import LeserKontaktFelder from './LeserKontaktFelder.svelte';
	import { istKollegium } from '../../leserArt.js';

	/**
	 * `formData` ist der $state-Proxy aus useStudentEditForm und wird hier direkt an den
	 * Feldern gebunden — kein $bindable: Gebunden werden die EIGENSCHAFTEN des Objekts,
	 * das Objekt selbst wird nie ersetzt. Ein $bindable verlangte vom Aufrufer ein
	 * `bind:`, und dort ist formData ein `const` aus der Hook-Destrukturierung.
	 * @type {{ formData: any, lusdVerknuepft?: boolean, kontoVorhanden?: boolean }}
	 */
	let { formData, lusdVerknuepft = false, kontoVorhanden = false } = $props();

	const kollege = $derived(istKollegium({ art: formData.art }));

	// Die Grenze verläuft in BEIDE Richtungen: Wer Schüler ist, kann nur Schüler bleiben;
	// wer keiner ist, wird keiner. Gesperrt ist deshalb immer die jeweils andere Seite.
	const gesperrteArten = $derived(kollege ? ['schueler'] : ['lehrkraft', 'liv']);

	// Der Stern markiert, was der Server verlangt — und er hängt an der ART, wie die
	// Pflicht selbst (chk_leser_schueler_pflichtfelder). Ohne ihn erfährt man erst beim
	// Speichern, dass ein Feld nicht leer bleiben darf.
	const schuelerPflicht = $derived(kollege ? '' : ' *');
</script>

<!-- ── Persönliche Daten ──────────────────────────────── -->
<section>
	<Abschnitt titel="Persönliche Daten" />

	<!-- Genau die Angabe, die in der Akte nirgends stand („hier steht nirgends ob jemand
	     ein Schüler, LiV, oder lehrer ist", 16.09.2026) — bei Littera steht sie
	     ebenfalls in den Stammdaten. -->
	<div class="mb-4">
		<LeserArtWahl bind:art={formData.art} gesperrt={gesperrteArten} />
		<p class="mt-1 text-xs text-on-surface-variant">
			{kollege
				? 'Zwischen Lehrkraft und LiV lässt sich wechseln. Ein Schüler kommt aus der LUSD — dorthin führt kein Weg.'
				: 'Ein Schüler kommt aus der LUSD und bleibt Schüler.'}
		</p>
	</div>

	<div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
		<Feld id="vorname" label="Vorname *" bind:value={formData.vorname} feld="font-semibold" />
		<Feld id="nachname" label="Nachname *" bind:value={formData.nachname} feld="font-semibold" />
		<Feld
			id="geburtsdatum"
			label="Geburtsdatum{schuelerPflicht}"
			type="date"
			bind:value={formData.geburtsdatum}
			hint={kollege ? undefined : 'Der Schlüssel, an dem der LUSD-Import wiedererkennt.'}
		/>
	</div>
</section>

<!-- ── Schuldaten ─────────────────────────────────────── -->
<section>
	<Abschnitt titel="Schuldaten" />

	<div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
		<Feld
			id="klasse"
			label="Klasse{schuelerPflicht}"
			bind:value={formData.klasse}
			feld="font-semibold"
			disabled={kollege}
			hint={kollege ? 'Nur ein Schüler hat eine Klasse.' : undefined}
		/>
		<Feld
			id="abgangsjahr"
			label="Abgangsjahr"
			type="number"
			bind:value={formData.abgaenger_jahr}
			feld="font-semibold"
			disabled={kollege}
			hint={kollege ? 'Gehört zur Klasse.' : undefined}
		/>
		<!-- Die LUSD-ID ist beim Schüler kontrolliert nachtragbar: nur setzbar, solange sie
		     leer ist (Waise adoptieren). Ist sie gesetzt, bleibt sie schreibgeschützt — der
		     Server lehnt Änderung und Leerung ohnehin ab. -->
		<Feld
			id="lusd_id"
			label="LUSD-ID"
			bind:value={formData.lusd_id}
			feld="font-mono"
			disabled={kollege || lusdVerknuepft}
			hint={kollege
				? 'Durch die LUSD kommen nur Schüler.'
				: lusdVerknuepft
					? 'Bereits mit der LUSD verknüpft — Änderung nur über den Import.'
					: 'Nachtragbar: verknüpft diesen Schüler mit der LUSD.'}
		/>
		<Feld
			id="barcode"
			label="Ausweisnummer{schuelerPflicht}"
			bind:value={formData.barcode_id}
			feld="font-mono"
			hint={kollege ? 'Leer lassen, solange kein Ausweis gedruckt ist.' : undefined}
		/>

		<!-- Kein Status-Dropdown: „status" ist ein abgeleiteter Lesewert
		     (aktiv/gesperrt/abgaenger aus ist_gesperrt/ist_abgaenger) ohne eigene
		     DB-Spalte. Sperren läuft übers Lock-Modal, Abgänger über das Abgangsjahr —
		     ein editierbares Feld hier wurde vom Backend still verworfen. -->
	</div>
</section>

<LeserKontaktFelder {formData} {kontoVorhanden} />
