<script>
	import { apiFetch } from './apiFetch.js';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import PageShell from './components/layout/PageShell.svelte';
	import Suchpille from './components/ui/Suchpille.svelte';
	import SuchZustand from './components/ui/SuchZustand.svelte';
	import { Search } from '@lucide/svelte';
	import AnliegenWidget from './components/portal/AnliegenWidget.svelte';
	import { erzeugeEigeneAnliegen } from './components/portal/eigeneAnliegen.svelte.js';
	import PortalTrefferkarte from './components/portal/PortalTrefferkarte.svelte';
	import PortalUeberblick from './components/portal/PortalUeberblick.svelte';
	import PortalLernmittel from './components/portal/PortalLernmittel.svelte';
	import PortalSchulbuecher from './components/portal/PortalSchulbuecher.svelte';
	import PortalLmfPlan from './components/portal/PortalLmfPlan.svelte';
	import Reiter from './components/ui/Reiter.svelte';
	import {
		erzeugeKlassensatzReservierung,
		erzeugeReservierungsListen
	} from './components/portal/klassensatzReservierung.svelte.js';
	/** @type {{ user: any }} */
	let { user } = $props();

	let reiter = $state('buecher');

	// Die eigenen Anliegen liegen an EINER Stelle (components/portal/eigeneAnliegen.svelte.js)
	// und nicht in den zwei Bauteilen, die sie zeigen: Der Zähler am Reiter, die
	// Startfläche und der Anliegen-Reiter sprechen sonst über denselben Zustand mit drei
	// Abrufen — und nach dem Absenden zeigte der Zähler noch den alten Stand.
	const eigeneAnliegen = erzeugeEigeneAnliegen();

	let searchQuery = $state('');
	let searchResults = $state.raw(/** @type {any[]} */ ([]));
	/** Der letzte Suchlauf ist gescheitert — dann steht hier kein Ergebnis, sondern nichts. */
	let suchfehler = $state(false);
	let isSearching = $state(false);

	let searchTimeout = /** @type {any} */ (null);

	// Warteschlange (alle) + eigene Reservierungen samt Bibliotheks-Notiz —
	// ausgelagert (Größen-Ratsche), Begründung und Zuschnitt in der Fabrik.
	const listen = erzeugeReservierungsListen();

	$effect(() => {
		listen.lade();
		eigeneAnliegen.lade();
	});

	// Formular-Zustand und Absenden je Titel — ausgelagert, Begründung dort.
	const reservierung = erzeugeKlassensatzReservierung(
		() => user,
		listen.warteschlangeFuer,
		listen.lade
	);

	$effect(() => {
		const q = searchQuery;
		clearTimeout(searchTimeout);
		if (q.trim().length < 2) {
			searchResults = [];
			return () => clearTimeout(searchTimeout);
		}
		searchTimeout = setTimeout(async () => {
			isSearching = true;
			try {
				// Bewusst der OPAC und nicht /api/search: Nur der OPAC rechnet die
				// Verfügbarkeit aus. /api/search liefert `BookTitle` — dort gibt es KEIN
				// Bestandsfeld, weshalb das Abzeichen unten still übersprungen wurde und
				// Lehrkräfte nie erfahren haben, ob ein Klassensatz überhaupt frei ist.
				//
				// Der OPAC passt auch fachlich: ausdrücklich nur Titel, Autor und Verfügbarkeit,
				// keine Ausleih- oder Personendaten — genau das, was eine Lehrkraft sehen darf.
				suchfehler = false;
				const res = await apiFetch(`/api/public/opac/suche?q=${encodeURIComponent(q)}`);
				if (res.ok) {
					const data = await res.json();
					searchResults = Array.isArray(data) ? data : (data.books ?? []);
				} else {
					// Sonst stünden die Treffer des vorigen Suchtextes unter der neuen
					// Eingabe (Sweep „verschluckte Fehlantwort", 06.09.2026).
					searchResults = [];
					suchfehler = true;
				}
			} catch {
				searchResults = [];
				suchfehler = true;
			} finally {
				isSearching = false;
			}
		}, 300);
		return () => clearTimeout(searchTimeout);
	});
</script>

<PageShell>
	<!-- Zwei Reiter statt zweier Aufgaben auf einer Fläche (Betreiber-Entscheidung
	     23.08.2026). Vorher stand oben ein namenloses Suchfeld, darunter ein 340-px-
	     Poster und ganz unten das Anliegen-Formular — dessen Felder dieselbe Pillenform
	     trugen wie die Suche, sodass „Welches Buch?" wie ein zweites Suchfeld aussah.
	     M3 kennt Reiter für genau diesen Fall: zwei gleichrangige Bereiche. -->
	<Reiter
		etikett="Portal-Bereiche"
		reiter={[
			// Drei gleichrangige Aufgaben (25.08.2026, Peters Ansage; „Bestand nach Jahrgang" am
			// 02.09.2026 gestrichen — Import-Default 5–10 machte die Gruppierung leer): „Lernmittel" stapelte
			// vorher zwei Listen mit eigenen Überschriften übereinander; und „Bücher &
			// Klassensätze" hieß fast so wie der Abschnitt „Klassensätze" darin — dreimal
			// dasselbe Wort für Suchen, Ansehen und den Menüpunkt.
			{ id: 'buecher', label: 'Suchen & Reservieren' },
			{ id: 'klassensaetze', label: 'Klassensätze' },
			// Schulbücher je Fach für die Fachsprecher (Peter, 03.09.2026).
			{ id: 'schulbuecher', label: 'Schulbücher' },
			// LMF-Plan für alle gleich statt Excel per Mail (Peter, 05.09.2026).
			{ id: 'lmfplan', label: 'LMF-Plan' },
			{ id: 'anliegen', label: 'Meine Anliegen', anzahl: eigeneAnliegen.offene }
		]}
		aktiv={reiter}
		onwahl={(id) => (reiter = id)}
	/>

	{#if reiter === 'buecher'}
		<!-- `mt-4`: Der Abstand Reiter→Pille ist im Haus 24 px (Huelle) + 16 px. Er fehlte
		     hier, die Pille begann bei 57 px statt bei 73. -->
		<div class="mt-4">
			<Suchpille
				id="portal-suchfeld"
				bind:wert={searchQuery}
				platzhalter="Titel, Autor oder ISBN eingeben …"
				etikett="Bücher für einen Klassensatz suchen"
				autofokus
				{nachlaufend}
			/>
		</div>

		{#if suchfehler}
			<p class="text-sm font-semibold text-error" role="alert">
				Die Suche ist fehlgeschlagen. Bitte erneut versuchen — es werden keine Treffer angezeigt,
				damit hier nichts Falsches steht.
			</p>
		{/if}

		{#if searchResults.length > 0}
			<div class="space-y-4">
				{#each searchResults as book (book.id ?? book.titel_id)}
					{@const titelId = book.id ?? book.titel_id}
					<PortalTrefferkarte
						{book}
						form={reservierung.form(titelId)}
						warteschlange={listen.warteschlangeFuer(titelId)}
						ontoggle={() => reservierung.toggle(titelId)}
						onsenden={() => reservierung.senden(titelId)}
					/>
				{/each}
			</div>
		{:else if searchQuery.trim().length >= 2 && !isSearching}
			<SuchZustand
				symbol={Search}
				titel="Keine Bücher gefunden"
				hinweis="Versuche es mit einem anderen Titel oder Autor."
			/>
		{:else if searchQuery.trim().length === 0}
			<PortalUeberblick reservierungen={listen.eigene} />
		{/if}
	{:else if reiter === 'klassensaetze'}
		<PortalLernmittel />
	{:else if reiter === 'schulbuecher'}
		<PortalSchulbuecher />
	{:else if reiter === 'lmfplan'}
		<PortalLmfPlan />
	{:else}
		<AnliegenWidget anliegen={eigeneAnliegen.liste} onaktualisiert={eigeneAnliegen.lade} />
	{/if}
</PageShell>

<!-- Der Ladepunkt sitzt IN der Pille, nicht darüber. Vorher lag er absolut positioniert
     bei right-4, während das Feld nur pr-4 Innenabstand hatte — ein langer Suchbegriff
     lief also unter den Punkt. -->
{#snippet nachlaufend()}
	{#if isSearching}
		<Ladekreis size="sm" />
	{/if}
{/snippet}
