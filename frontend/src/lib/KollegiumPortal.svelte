<script>
	import { apiFetch } from './apiFetch.js';
	import PageShell from './components/layout/PageShell.svelte';
	import Suchpille from './components/ui/Suchpille.svelte';
	import SuchZustand from './components/ui/SuchZustand.svelte';
	import { Search } from '@lucide/svelte';
	import AnliegenWidget from './components/portal/AnliegenWidget.svelte';
	import PortalTrefferkarte from './components/portal/PortalTrefferkarte.svelte';
	import PortalUeberblick from './components/portal/PortalUeberblick.svelte';
	import PortalLernmittel from './components/portal/PortalLernmittel.svelte';
	import Reiter from './components/ui/Reiter.svelte';
	import {
		erzeugeKlassensatzReservierung,
		erzeugeReservierungsListen
	} from './components/portal/klassensatzReservierung.svelte.js';
	/** @type {{ user: any }} */
	let { user } = $props();

	let reiter = $state('buecher');

	/**
	 * Die eigenen Anliegen liegen HIER und nicht in den zwei Bauteilen, die sie zeigen:
	 * Der Zähler am Reiter, die Startfläche und der Anliegen-Reiter sprechen sonst über
	 * denselben Zustand mit drei Abrufen — und nach dem Absenden zeigte der Zähler noch
	 * den alten Stand.
	 * @type {{ id: string, art: string, titel_text: string, klasse: string, kommentar?: string, erstellt_am: string, erledigt_am?: string, erledigt_notiz?: string }[]}
	 */
	let eigeneAnliegen = $state([]);
	const offeneAnliegen = $derived(eigeneAnliegen.filter((a) => !a.erledigt_am).length);

	async function ladeAnliegen() {
		try {
			const res = await apiFetch('/api/anliegen/eigene');
			const daten = res.ok ? await res.json() : [];
			if (Array.isArray(daten)) eigeneAnliegen = daten;
		} catch {
			/* Zusatzinfo — ohne sie bleibt das Portal benutzbar */
		}
	}

	let searchQuery = $state('');
	let searchResults = $state.raw(/** @type {any[]} */ ([]));
	let isSearching = $state(false);

	let searchTimeout = /** @type {any} */ (null);

	// Warteschlange (alle) + eigene Reservierungen samt Bibliotheks-Notiz —
	// ausgelagert (Größen-Ratsche), Begründung und Zuschnitt in der Fabrik.
	const listen = erzeugeReservierungsListen();

	$effect(() => {
		listen.lade();
		ladeAnliegen();
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
				const res = await apiFetch(`/api/public/opac/suche?q=${encodeURIComponent(q)}`);
				if (res.ok) {
					const data = await res.json();
					searchResults = Array.isArray(data) ? data : (data.books ?? []);
				}
			} catch {
				/* ignore */
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
			{ id: 'anliegen', label: 'Meine Anliegen', anzahl: offeneAnliegen }
		]}
		aktiv={reiter}
		onwahl={(id) => (reiter = id)}
	/>

	{#if reiter === 'buecher'}
		<Suchpille
			id="portal-suchfeld"
			bind:wert={searchQuery}
			platzhalter="Titel, Autor oder ISBN eingeben …"
			etikett="Bücher für einen Klassensatz suchen"
			autofokus
			{nachlaufend}
		/>

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
	{:else}
		<AnliegenWidget anliegen={eigeneAnliegen} onaktualisiert={ladeAnliegen} />
	{/if}
</PageShell>

<!-- Der Ladepunkt sitzt IN der Pille, nicht darüber. Vorher lag er absolut positioniert
     bei right-4, während das Feld nur pr-4 Innenabstand hatte — ein langer Suchbegriff
     lief also unter den Punkt. -->
{#snippet nachlaufend()}
	{#if isSearching}
		<div
			class="shrink-0 w-4 h-4 border-2 border-primary/40 border-t-primary rounded-full animate-spin"
			aria-hidden="true"
		></div>
	{/if}
{/snippet}
