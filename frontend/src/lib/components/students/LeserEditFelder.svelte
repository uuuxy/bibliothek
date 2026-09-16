<!-- @component LeserEditFelder — die Eingabefelder der Leserakte. EINE Maske für jeden.

     Bis zum 16.09.2026 war dieses Formular ein Schüler-Formular: Klasse, Abgangsjahr,
     LUSD-ID und Eltern-E-Mail standen fest darin. Ein Kollege kam gar nicht erst hierher
     (seine Akte hatte keinen Bearbeiten-Knopf) — und hätte er es, wäre er an der leeren
     Klasse gescheitert, die der Server als Pflichtfeld abweist. Peter am 16.09.2026:
     „die Leute die bereits eine Rolle haben stehen zwar in der ausleihliste, ich kann
     dort aber keine adressedaten etc nachtragen."

     Mein erster Versuch blendete die Felder je nach Art ein und aus — zwei Masken. Peter:
     „warum eine andere maske als bei schülern? das ist doch schon wieder viel zu
     kompliziert." Er hat recht: Zwingend ist nur EIN Unterschied, der Rest war Vorsicht.

     Deshalb steht hier für jeden dasselbe an derselben Stelle. Abgeschaltet statt
     versteckt ist nur, was wirklich nicht gehen darf:

       - LUSD-ID: Die Datenbank verbietet sie einem Kollegen
         (chk_leser_nur_schueler_werden_abgaenger). Durch die LUSD kommen nur Schüler.
       - Klasse und Abgangsjahr: „eine klasse muss ja keinem lehrer/liv zugeordnet
         werden" (Peter). Das Abgangsjahr hängt an der Klasse — der Server leitet es aus
         ihr ab (calculateAbgaengerJahr), also gehören beide zusammen.
       - Die Art über die Schüler-Grenze: Ein Schüler bleibt Schüler.

     Alles andere — Geburtsdatum, Ausweisnummer, Postanschrift, Eltern-E-Mail — steht
     jedem offen und bleibt beim Kollegen einfach leer. -->
<script>
	import Feld from '../ui/Feld.svelte';
	import LeserArtWahl from './LeserArtWahl.svelte';
	import { istKollegium } from '../../leserArt.js';

	/**
	 * `formData` ist der $state-Proxy aus useStudentEditForm und wird hier direkt an den
	 * Feldern gebunden — kein $bindable: Gebunden werden die EIGENSCHAFTEN des Objekts,
	 * das Objekt selbst wird nie ersetzt. Ein $bindable verlangte vom Aufrufer ein
	 * `bind:`, und dort ist formData ein `const` aus der Hook-Destrukturierung.
	 * @type {{ formData: any, lusdVerknuepft?: boolean }}
	 */
	let { formData, lusdVerknuepft = false } = $props();

	const kollege = $derived(istKollegium({ art: formData.art }));

	// Die Grenze verläuft in BEIDE Richtungen: Wer Schüler ist, kann nur Schüler bleiben;
	// wer keiner ist, wird keiner. Gesperrt ist deshalb immer die jeweils andere Seite.
	const gesperrteArten = $derived(kollege ? ['schueler'] : ['lehrkraft', 'liv']);
</script>

{#snippet abschnitt(titel, punktfarbe)}
	<h3 class="mb-4 flex items-center gap-2 text-base font-medium text-slate-500">
		<div class="h-2.5 w-2.5 rounded-full {punktfarbe}"></div>
		{titel}
	</h3>
{/snippet}

<!-- ── Persönliche Daten ──────────────────────────────── -->
<section>
	{@render abschnitt('Persönliche Daten', 'bg-slate-300')}

	<div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
		<Feld id="vorname" label="Vorname" bind:value={formData.vorname} feld="font-semibold" />
		<Feld id="nachname" label="Nachname" bind:value={formData.nachname} feld="font-semibold" />
		<Feld id="geburtsdatum" label="Geburtsdatum" type="date" bind:value={formData.geburtsdatum} />
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
	</div>

	<!-- Genau die Angabe, die in der Akte nirgends stand („hier steht nirgends ob jemand
	     ein Schüler, LiV, oder lehrer ist", Peter am 16.09.2026) — bei Littera steht sie
	     ebenfalls in den Stammdaten. -->
	<div class="mt-4">
		<LeserArtWahl bind:art={formData.art} gesperrt={gesperrteArten} />
		<p class="mt-1 text-xs text-on-surface-variant">
			{kollege
				? 'Zwischen Lehrkraft und LiV lässt sich wechseln. Ein Schüler kommt aus der LUSD — dorthin führt kein Weg.'
				: 'Ein Schüler kommt aus der LUSD und bleibt Schüler.'}
		</p>
	</div>
</section>

<!-- ── Schuldaten ─────────────────────────────────────── -->
<section>
	{@render abschnitt('Schuldaten', 'bg-slate-300')}

	<div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
		<Feld
			id="klasse"
			label="Klasse"
			bind:value={formData.klasse}
			feld="font-semibold"
			disabled={kollege}
			hint={kollege ? 'Eine Klasse hat nur ein Schüler.' : undefined}
		/>
		<Feld
			id="barcode"
			label="Ausweisnummer"
			bind:value={formData.barcode_id}
			feld="font-mono"
			hint={kollege ? 'Leer lassen, solange kein Ausweis gedruckt ist.' : undefined}
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

		<!-- Kein Status-Dropdown: „status" ist ein abgeleiteter Lesewert
		     (aktiv/gesperrt/abgaenger aus ist_gesperrt/ist_abgaenger) ohne eigene
		     DB-Spalte. Sperren läuft übers Lock-Modal, Abgänger über das Abgangsjahr —
		     ein editierbares Feld hier wurde vom Backend still verworfen. -->
	</div>
</section>

<!-- ── Kontaktdaten ────────────────────────────────────── -->
<section>
	{@render abschnitt('Kontaktdaten', 'bg-blue-400')}

	<!-- EIN flaches Raster, keine Hülle: Eine <div>-Hülle spannt eine Rasterzeile, das
	     Feld braucht drei (Subgrid) — es rutscht aus der Reihe (feld-huellen.test.js). -->
	<div class="grid grid-cols-4 gap-4 md:grid-cols-8">
		<Feld
			id="strasse"
			label="Straße"
			class="col-span-3"
			bind:value={formData.strasse}
			placeholder="Musterstraße"
		/>
		<Feld id="hausnummer" label="Nr." bind:value={formData.hausnummer} placeholder="12a" />
		<Feld
			id="plz"
			label="PLZ"
			class="col-span-2"
			bind:value={formData.plz}
			placeholder="12345"
			maxlength={5}
			feld="font-mono"
		/>
		<Feld
			id="ort"
			label="Ort"
			class="col-span-2"
			bind:value={formData.ort}
			placeholder="Musterstadt"
		/>
		<Feld
			id="email"
			label="Eltern E-Mail"
			class="col-span-4"
			type="email"
			bind:value={formData.eltern_email}
			placeholder="eltern@schule.de"
		/>
	</div>
</section>
