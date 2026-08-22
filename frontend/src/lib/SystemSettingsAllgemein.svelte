<script>
	import { apiPut } from './apiFetch.js';
	import { zahlOderUnveraendert } from './settingsWerte.js';
	import { toastStore } from './stores/toastStore.svelte.js';
	import SettingField from './components/settings/SettingField.svelte';
	import Switch from './components/ui/Switch.svelte';

	/**
	 * @typedef {Object} Props
	 * @property {boolean} ferienLeseclubAktiv
	 * @property {string} ferienLeseclubZieldatum
	 * @property {string} lmfStichtag
	 * @property {number} maxAusleihenSchueler
	 * @property {number} fristBuchTage
	 * @property {number} fristMedienTage
	 * @property {number} maxOverdueDays
	 * @property {number} maxOverdueItems
	 * @property {boolean} bestellbedarfWarnungAktiv
	 * @property {number} bestellbedarfSchwelle
	 * @property {boolean} preiseErfassen
	 * @property {string} schuleName
	 * @property {string} schuleStrasse
	 * @property {string} schulePLZ
	 * @property {string} schuleOrt
	 * @property {string} etikettEigentumsvermerk
	 * @property {string} oeffentlicheAdresse
	 * @property {string} alarmEmpfaenger
	 * @property {number} lesehistorieTage
	 * @property {number} lesehistorieLernmittelTage
	 * @property {number} thekeLeerenMinuten
	 * @property {number} sperreMinuten
	 * @property {() => void | Promise<void>} [onSaved] Nach dem Speichern neu laden (leer gelassene Zahlenfelder zeigen sonst nicht den gespeicherten Wert)
	 */

	/** @type {Props} */
	let {
		ferienLeseclubAktiv = $bindable(),
		ferienLeseclubZieldatum = $bindable(),
		lmfStichtag = $bindable(),
		maxAusleihenSchueler = $bindable(),
		fristBuchTage = $bindable(),
		fristMedienTage = $bindable(),
		maxOverdueDays = $bindable(),
		maxOverdueItems = $bindable(),
		bestellbedarfWarnungAktiv = $bindable(),
		bestellbedarfSchwelle = $bindable(),
		preiseErfassen = $bindable(),
		schuleName = $bindable(),
		schuleStrasse = $bindable(),
		schulePLZ = $bindable(),
		schuleOrt = $bindable(),
		etikettEigentumsvermerk = $bindable(),
		oeffentlicheAdresse = $bindable(),
		alarmEmpfaenger = $bindable(),
		lesehistorieTage = $bindable(),
		lesehistorieLernmittelTage = $bindable(),
		thekeLeerenMinuten = $bindable(),
		sperreMinuten = $bindable(),
		onSaved = undefined
	} = $props();

	let saving = $state(false);

	async function saveSettings() {
		saving = true;
		try {
			await apiPut('/api/einstellungen', {
				ferien_leseclub_aktiv: ferienLeseclubAktiv,
				ferien_leseclub_zieldatum: ferienLeseclubZieldatum || null,
				lmf_stichtag: lmfStichtag || '07-31',
				max_ausleihen_schueler: maxAusleihenSchueler,
				frist_buch_tage: fristBuchTage,
				frist_medien_tage: fristMedienTage,
				max_overdue_days: maxOverdueDays,
				max_overdue_items: maxOverdueItems,
				bestellbedarf_warnung_aktiv: bestellbedarfWarnungAktiv,
				bestellbedarf_schwelle: bestellbedarfSchwelle,
				preise_erfassen: preiseErfassen,
				schule_name: schuleName,
				schule_strasse: schuleStrasse,
				schule_plz: schulePLZ,
				schule_ort: schuleOrt,
				etikett_eigentumsvermerk: etikettEigentumsvermerk,
				// Immer mitgeschickt, auch leer: Für dieses Feld heisst leer ausdrücklich
				// "abschalten" (dann verschickt das System keine Bestätigungs-Links mehr).
				oeffentliche_adresse: oeffentlicheAdresse,
				alarm_empfaenger: alarmEmpfaenger,
				// 0 heisst hier ausdrücklich "aus" und ist ein Wert — aber nur die GETIPPTE 0.
				// Ein geleertes Feld geht als null raus (= unverändert lassen); vorher machte
				// `Number(null) || 0` daraus still ein "aus" (Prüfung 22.08.2026, A4).
				lesehistorie_tage: zahlOderUnveraendert(lesehistorieTage),
				lesehistorie_lernmittel_tage: zahlOderUnveraendert(lesehistorieLernmittelTage),
				theke_leeren_minuten: zahlOderUnveraendert(thekeLeerenMinuten),
				sperre_minuten: zahlOderUnveraendert(sperreMinuten)
			});
			toastStore.addToast('Einstellungen gespeichert.', 'success');
			if (onSaved) await onSaved();
		} catch {
			// Toast already shown by apiPut
		}
		saving = false;
	}
</script>

<!-- Flach & edge-to-edge: logische Blöcke per Abstand + feiner Trennlinie statt Kacheln -->
<div class="space-y-12 max-w-5xl">
	{#snippet sectionHeader(title, description)}
		<div>
			<h3 class="text-lg font-bold text-slate-900">{title}</h3>
			{#if description}
				<p class="text-sm text-slate-600 mt-1.5 leading-relaxed max-w-2xl">{description}</p>
			{/if}
		</div>
	{/snippet}

	<!-- Schule.
	     Steht ganz oben, weil es Stammdatum ist: Der Name erscheint auf jedem Buchetikett
	     und im Briefkopf von Mahnung, Bestellung und allen Berichten. Bis zum 04.08.2026
	     ließ er sich nirgends eintragen — die Felder lagen nur in der Datenbank, und auf
	     dem Etikett stand ersatzweise fest "Schulbibliothek". -->
	<section class="border-b border-slate-200 pb-8">
		{@render sectionHeader(
			'Schule',
			'Erscheint auf jedem Buchetikett, auf dem Schülerausweis und als Briefkopf in Mahnungen, Bestellungen und Berichten. Grauer Text ist ein Beispiel, kein gespeicherter Wert — steht hier nichts Schwarzes, ist das Feld leer. Ein leer gelassenes Feld bleibt beim Speichern unverändert; bereits gespeicherte Angaben werden nicht gelöscht.'
		)}
		<!-- Platzhalter bewusst als Musterdaten, NICHT als die echten Schuldaten.
		     Bis zum 06.08.2026 stand hier "Philipp-Reis-Schule, Friedrichsdorf",
		     "Hoher Weg 29", "61381" und "Friedrichsdorf" — also genau das, was
		     eingetragen gehört. Zusammen mit dem Hinweis "leer lassen ändert nichts"
		     las sich das Formular wie ausgefüllt, und niemand trug etwas ein. In der
		     Datenbank standen bis dahin leere Strings, und der Schülerausweis zeigte
		     weiter "STÄDTISCHES GYMNASIUM MUSTERSTADT" — die Selbstheilung
		     (wendeSchulstammdatenAn) ersetzt den Musterkopf nur, wenn ein Schulname
		     hinterlegt IST. Ein Platzhalter, der aussieht wie ein Wert, ist kein Hinweis,
		     sondern eine Falle. Die Muster hier spiegeln jetzt den Ausweis-Platzhalter,
		     damit die Verbindung erkennbar ist. -->
		<div class="mt-6 grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-5">
			<SettingField
				bind:value={schuleName}
				label="Name der Schule"
				type="text"
				maxlength={120}
				placeholder="z. B. Städtisches Gymnasium Musterstadt"
				hint="Erste Zeile auf dem Buchetikett und Kopfzeile des Schülerausweises."
			/>
			<SettingField
				bind:value={etikettEigentumsvermerk}
				label="Eigentumsvermerk"
				type="text"
				maxlength={80}
				placeholder="Eigentum des Landes Hessen"
				hint="Letzte Zeile auf dem Buchetikett. Leer lassen für die Vorgabe."
			/>
			<SettingField
				bind:value={schuleStrasse}
				label="Straße und Hausnummer"
				type="text"
				maxlength={120}
				placeholder="z. B. Musterstraße 12"
			/>
			<!-- Keine Kosmetik: Aus dieser Adresse entsteht der Bestätigungs-Link, den
			     Lieferanten mit der Bestellmail bekommen. Fehlt sie, geht die Bestellung
			     ohne Link raus — der Server kann sie nicht erraten, weil er hinter einem
			     Reverse-Proxy nur seinen internen Namen sieht. -->
			<SettingField
				bind:value={oeffentlicheAdresse}
				label="Öffentliche Adresse"
				type="text"
				maxlength={200}
				placeholder="https://bibliothek.schule.de"
				hint="Adresse, unter der Lieferanten das System von außen erreichen. Grundlage des Bestätigungs-Links in der Bestellmail; leer = keine Links verschicken."
			/>
			<!-- Betreiber-Wunsch 17.08.2026 nach dem Alarm-Mail-Vorfall: Der Kritisch-
			     Alarm der Selbstprüfung ging an ALLE aktiven Admin-Konten — auch an
			     eines, das niemand kannte. Hier lässt sich der Empfängerkreis festlegen;
			     leer bleibt der sichere Rückfall (alle aktiven Admins), denn ein Alarm,
			     der niemanden erreicht, ist keiner. -->
			<SettingField
				bind:value={alarmEmpfaenger}
				label="Alarm-Empfänger (Betriebsbereitschaft)"
				type="text"
				maxlength={300}
				placeholder="pflasch@philipp-reis-schule.de, it@schule.de"
				hint="Kritisch-Alarme der Selbstprüfung gehen nur an diese Adressen (mehrere mit Komma). Leer = alle aktiven Admin-Konten."
			/>
			<div class="grid grid-cols-3 gap-4">
				<SettingField
					bind:value={schulePLZ}
					label="PLZ"
					type="text"
					maxlength={10}
					placeholder="12345"
				/>
				<div class="col-span-2">
					<SettingField
						bind:value={schuleOrt}
						label="Ort"
						type="text"
						maxlength={80}
						placeholder="Musterstadt"
					/>
				</div>
			</div>
		</div>
	</section>

	<!-- Ferien-Leseclub -->
	<section class="border-b border-slate-200 pb-8">
		<div class="flex items-start justify-between gap-4">
			{@render sectionHeader(
				'Ferien-Leseclub',
				'Wenn aktiv, erhalten alle neuen Ausleihen pauschal das unten definierte Ferienende als Rückgabefrist. Die Standardfristen werden überschrieben.'
			)}
			<button
				type="button"
				onclick={() => (ferienLeseclubAktiv = !ferienLeseclubAktiv)}
				class="relative inline-flex h-8 w-14 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 {ferienLeseclubAktiv
					? 'bg-emerald-500'
					: 'bg-slate-200'}"
				role="switch"
				aria-checked={ferienLeseclubAktiv}
				aria-label="Ferien-Leseclub umschalten"
			>
				<span
					class="pointer-events-none inline-block h-7 w-7 transform rounded-full bg-white shadow-sm transition duration-200 ease-in-out {ferienLeseclubAktiv
						? 'translate-x-6'
						: 'translate-x-0'}"
				></span>
			</button>
		</div>

		{#if ferienLeseclubAktiv}
			<div class="mt-6 max-w-xs">
				<SettingField
					bind:value={ferienLeseclubZieldatum}
					label="Ferienende (Rückgabezieldatum)"
					type="date"
					hint="Alle Ausleihen erhalten dieses Datum als Rückgabefrist."
				/>
			</div>
		{/if}
	</section>

	<!-- Rückgabefristen & Ausleih-Limits -->
	<section class="border-b border-slate-200 pb-8">
		{@render sectionHeader('Rückgabefristen & Ausleih-Limits', '')}
		<div class="mt-8 grid grid-cols-2 md:grid-cols-4 gap-x-12 gap-y-10">
			<SettingField bind:value={fristBuchTage} label="Tage / Buch" min={1} max={365} />
			<SettingField bind:value={fristMedienTage} label="Tage / Medien" min={1} max={365} />
			<SettingField
				bind:value={lmfStichtag}
				label="LMF (MM-TT)"
				type="text"
				placeholder="07-31"
				pattern={'\\d{2}-\\d{2}'}
				maxlength={5}
			/>
			<SettingField bind:value={maxAusleihenSchueler} label="Max Ausleihen" min={1} max={50} />
		</div>
	</section>

	<!-- Sperr-Automatik -->
	<section class="border-b border-slate-200 pb-8">
		{@render sectionHeader(
			'Sperr-Automatik (Mahnwesen)',
			'Automatische Ausleihsperre am Kiosk für Schüler mit überfälligen Medien. Ausgenommen sind Geräte/Dauerleihen (z.B. Laptops).'
		)}
		<div class="mt-8 grid grid-cols-2 gap-x-12 gap-y-10 max-w-xl">
			<SettingField bind:value={maxOverdueDays} label="Tage bis Sperre" min={0} max={365} />
			<SettingField bind:value={maxOverdueItems} label="Ab x Medien sperren" min={1} max={50} />
		</div>
	</section>

	<!-- Bestellbedarf-Warnung -->
	<section class="border-b border-slate-200 pb-8">
		<div class="flex items-start justify-between gap-4">
			{@render sectionHeader(
				'Bestellbedarf-Warnung',
				'Meldet Schulbücher (LMF), deren eigener Bestand unter die Schwelle fällt. Aus = keine Bestellbedarf-Liste. Ersetzt den früheren pauschalen Meldebestand.'
			)}
			<button
				type="button"
				onclick={() => (bestellbedarfWarnungAktiv = !bestellbedarfWarnungAktiv)}
				class="relative inline-flex h-8 w-14 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 {bestellbedarfWarnungAktiv
					? 'bg-emerald-500'
					: 'bg-slate-200'}"
				role="switch"
				aria-checked={bestellbedarfWarnungAktiv}
				aria-label="Bestellbedarf-Warnung umschalten"
			>
				<span
					class="pointer-events-none inline-block h-7 w-7 transform rounded-full bg-white shadow-sm transition duration-200 ease-in-out {bestellbedarfWarnungAktiv
						? 'translate-x-6'
						: 'translate-x-0'}"
				></span>
			</button>
		</div>

		{#if bestellbedarfWarnungAktiv}
			<div class="mt-6 max-w-xs">
				<SettingField
					bind:value={bestellbedarfSchwelle}
					label="Warnen unter x Exemplaren"
					min={1}
					max={50}
					hint="Ein Titel gilt als Bestellbedarf, wenn er weniger als x eigene (nicht ausgesonderte) Exemplare hat."
				/>
			</div>
		{/if}
	</section>

	<!-- Datenschutz & Sitzung (docs/datenschutz_offene_punkte.md A1/A4). Zwei Fristen,
	     weil zwei Verarbeitungstätigkeiten: Schülerbücherei kurz (HBDI-Muster: löschen,
	     sobald nicht mehr notwendig), Lernmittel lang (Bestandskartei weist Ausleihe UND
	     Rücklauf nach, Schadensersatz läuft über die Schulaufsicht). Der nächtliche Job
	     trennt die Ausleihe vom Schüler; der Vorgang selbst bleibt für die Statistik. -->
	<section class="border-b border-outline-variant pb-8">
		{@render sectionHeader(
			'Datenschutz & Sitzung',
			'Lesehistorie: Tage nach der Rückgabe, nach denen eine Ausleihe nicht mehr dem Schüler zugeordnet ist (sie zählt weiter in der Statistik, nur ohne Namen). Offene Schadensfälle halten die Zuordnung. Sitzung: Minuten ohne Bedienung, bis die Theke den geladenen Schüler fallen lässt bzw. der Sperrbildschirm kommt. Eine getippte 0 = aus; ein leer gelassenes Feld bleibt beim Speichern unverändert.'
		)}
		<div class="mt-8 grid grid-cols-2 md:grid-cols-4 gap-x-12 gap-y-10">
			<SettingField
				bind:value={lesehistorieTage}
				label="Lesehistorie Schülerbücherei (Tage)"
				min={0}
				max={3650}
				hint="Vorgabe 90."
			/>
			<SettingField
				bind:value={lesehistorieLernmittelTage}
				label="Lesehistorie Lernmittel (Tage)"
				min={0}
				max={3650}
				hint="Vorgabe 730 (zwei Schuljahre)."
			/>
			<SettingField
				bind:value={thekeLeerenMinuten}
				label="Theke leeren nach (Min.)"
				min={0}
				max={1440}
				hint="Vorgabe 5."
			/>
			<SettingField
				bind:value={sperreMinuten}
				label="Sperrbildschirm nach (Min.)"
				min={0}
				max={1440}
				hint="Vorgabe 15. Entsperren mit dem eigenen Passwort."
			/>
		</div>
	</section>

	<!-- Preiserfassung: entscheidet, ob das Bestellwesen ueberhaupt mit Geld arbeitet. -->
	<section class="border-b border-slate-200 pb-8">
		<div class="flex items-start justify-between gap-4">
			{@render sectionHeader(
				'Preise im Bestellwesen',
				'An = Preisfeld im Warenkorb, Betragsspalten in der Bestellhistorie, Berichte mit Summen. Aus = die Bestellhistorie und alle Berichte zählen Exemplare statt Euro. Bereits erfasste Beträge bleiben gespeichert und erscheinen wieder, sobald der Schalter zurückgelegt wird.'
			)}
			<Switch bind:checked={preiseErfassen} label="Preise im Bestellwesen umschalten" />
		</div>
	</section>

	<div class="flex justify-end">
		<button
			onclick={saveSettings}
			disabled={saving}
			class="px-8 py-3 bg-slate-900 hover:bg-slate-800 text-white font-bold text-sm rounded-full transition-colors cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed shadow-sm"
		>
			{saving ? 'Wird gespeichert...' : 'Globale Einstellungen speichern'}
		</button>
	</div>
</div>
