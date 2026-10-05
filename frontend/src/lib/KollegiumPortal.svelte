<script>
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import PageShell from './components/layout/PageShell.svelte';
	import Suchpille from './components/ui/Suchpille.svelte';
	import SuchZustand from './components/ui/SuchZustand.svelte';
	import { Search } from '@lucide/svelte';
	import PortalMeldungen from './components/portal/PortalMeldungen.svelte';
	import { erzeugeEigeneAnliegen } from './components/portal/eigeneAnliegen.svelte.js';
	import PortalTrefferkarte from './components/portal/PortalTrefferkarte.svelte';
	import PortalUeberblick from './components/portal/PortalUeberblick.svelte';
	import PortalLernmittel from './components/portal/PortalLernmittel.svelte';
	import PortalSchulbuecher from './components/portal/PortalSchulbuecher.svelte';
	import PortalLmfPlan from './components/portal/PortalLmfPlan.svelte';
	import Reiter from './components/ui/Reiter.svelte';
	import FilterChips from './components/ui/FilterChips.svelte';
	import { erzeugePortalSuche } from './components/portal/portalSuche.svelte.js';
	import {
		erzeugeKlassensatzReservierung,
		erzeugeReservierungsListen
	} from './components/portal/klassensatzReservierung.svelte.js';
	import { erzeugeProblemMeldung, OHNE_BUCH } from './components/portal/problemMeldung.svelte.js';
	/** @type {{ user: any }} */
	let { user } = $props();

	let reiter = $state('buecher');

	// Die eigenen Anliegen: Die Liste steht unter der Suche, und eine Meldung am Treffer
	// liest sie neu.
	const eigeneAnliegen = erzeugeEigeneAnliegen();

	// Suchtext, Filter nach Schlagwort und Treffer — ausgelagert, Begründung in der Fabrik.
	const suche = erzeugePortalSuche();

	// Warteschlange (alle) + eigene Reservierungen samt Bibliotheks-Notiz —
	// ausgelagert (Größen-Ratsche), Begründung und Zuschnitt in der Fabrik.
	const listen = erzeugeReservierungsListen();

	$effect(() => {
		listen.lade();
		eigeneAnliegen.lade();
		suche.ladeFilter();
	});

	// Formular-Zustand und Absenden je Titel — ausgelagert, Begründung dort.
	const reservierung = erzeugeKlassensatzReservierung(
		() => user,
		listen.warteschlangeFuer,
		listen.lade
	);

	// „Problem melden" je Treffer und einmal ohne Buch; nach dem Absenden liest das Portal
	// die eigenen Anliegen neu.
	const meldung = erzeugeProblemMeldung(() => eigeneAnliegen.lade());

	// Je Karte ist höchstens ein Formular offen: Das eine schließt das andere.
	/** @param {string} titelId */
	function reserviereAmTreffer(titelId) {
		meldung.schliesse(titelId);
		reservierung.toggle(titelId);
	}
	/** @param {string} titelId */
	function meldeAmTreffer(titelId) {
		if (meldung.form(titelId).open) return meldung.schliesse(titelId);
		if (reservierung.form(titelId).open) reservierung.toggle(titelId);
		meldung.oeffne(titelId);
	}
</script>

<PageShell>
	<!-- Vier Reiter. Der erste nimmt alles auf, was eine Lehrkraft an die Bibliothek schickt —
	     suchen, reservieren, ein Problem melden — und zeigt, was daraus geworden ist; die
	     drei anderen sind Seiten zum Nachschlagen. M3 rät von mehr als vier Reitern ab. -->
	<Reiter
		etikett="Portal-Bereiche"
		reiter={[
			{ id: 'buecher', label: 'Suchen & Reservieren' },
			{ id: 'klassensaetze', label: 'Klassensätze' },
			// Schulbücher je Fach für die Fachsprecher.
			{ id: 'schulbuecher', label: 'Schulbücher' },
			// Derselbe LMF-Plan für alle.
			{ id: 'lmfplan', label: 'LMF-Plan' }
		]}
		aktiv={reiter}
		onwahl={(id) => (reiter = id)}
	/>

	{#if reiter === 'buecher'}
		<!-- `mt-4`: Der Abstand Reiter→Pille ist im Haus 24 px (Hülle) + 16 px. Die Filter nach
		     Schlagwort stehen darunter wie die Filterzeile der Leserdatei (gap-3); ohne
		     markierte Wörter entfällt die Zeile. -->
		<div class="mt-4 flex flex-col gap-3">
			<Suchpille
				id="portal-suchfeld"
				bind:wert={suche.text}
				platzhalter="Titel, Autor oder ISBN eingeben …"
				etikett="Bücher für einen Klassensatz suchen"
				autofokus
				{nachlaufend}
			/>
			{#if suche.filter.length > 0}
				<FilterChips
					optionen={suche.filter.map((f) => ({ wert: f.id, text: f.wort }))}
					wert={suche.schlagwort}
					onwahl={(w) => (suche.schlagwort = w)}
					etikett="Nach Schlagwort filtern"
				/>
			{/if}
		</div>

		{#if suche.fehler}
			<p class="text-sm font-semibold text-error" role="alert">
				Die Suche ist fehlgeschlagen. Bitte erneut versuchen — es werden keine Treffer angezeigt,
				damit hier nichts Falsches steht.
			</p>
		{/if}

		{#if suche.treffer.length > 0}
			{#if suche.gesamt > suche.treffer.length}
				<p class="text-sm text-on-surface-variant">
					Gezeigt werden {suche.treffer.length} von {suche.gesamt} Treffern — ein Suchwort grenzt ein.
				</p>
			{/if}
			<div class="space-y-4">
				{#each suche.treffer as book (book.id ?? book.titel_id)}
					{@const titelId = book.id ?? book.titel_id}
					<PortalTrefferkarte
						{book}
						form={reservierung.form(titelId)}
						meldung={meldung.form(titelId)}
						warteschlange={listen.warteschlangeFuer(titelId)}
						ontoggle={() => reserviereAmTreffer(titelId)}
						onsenden={() => reservierung.senden(titelId)}
						onmelden={() => meldeAmTreffer(titelId)}
						onmeldungsenden={() => meldung.senden(titelId, book.titel ?? book.title)}
						onmeldungabbrechen={() => meldung.schliesse(titelId)}
					/>
				{/each}
			</div>
		{:else if suche.aktiv && !suche.laedt && !suche.fehler}
			<!-- Nicht bei einem Fehler: Dann ist nichts gefunden, weil nicht gesucht wurde. -->
			<SuchZustand
				symbol={Search}
				titel="Keine Bücher gefunden"
				hinweis={suche.schlagwort
					? 'Versuche es mit einem anderen Suchwort oder nimm den Filter zurück.'
					: 'Versuche es mit einem anderen Titel oder Autor.'}
			/>
		{:else if suche.leer}
			<!-- Solange nichts gesucht wird: was die Lehrkraft geschickt hat und was daraus
			     geworden ist, darunter „Problem melden" ohne Buch. -->
			<PortalUeberblick reservierungen={listen.eigene} />
			<PortalMeldungen
				anliegen={eigeneAnliegen.liste}
				form={meldung.form(OHNE_BUCH)}
				onoeffnen={() => meldung.oeffne(OHNE_BUCH)}
				onsenden={() => meldung.senden(OHNE_BUCH)}
				onabbrechen={() => meldung.schliesse(OHNE_BUCH)}
				onaktualisiert={eigeneAnliegen.lade}
				ladefehler={eigeneAnliegen.fehler}
			/>
		{/if}
	{:else if reiter === 'klassensaetze'}
		<PortalLernmittel />
	{:else if reiter === 'schulbuecher'}
		<PortalSchulbuecher />
	{:else}
		<PortalLmfPlan />
	{/if}
</PageShell>

<!-- Der Ladepunkt sitzt in der Pille: Absolut darüber gelegt, liefe ein langer Suchbegriff
     unter ihn. -->
{#snippet nachlaufend()}
	{#if suche.laedt}
		<Ladekreis size="sm" />
	{/if}
{/snippet}
