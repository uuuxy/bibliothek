<script>
	import { apiFetch } from './apiFetch.js';
	import PageShell from './components/layout/PageShell.svelte';
	import Suchpille from './components/ui/Suchpille.svelte';
	import SuchZustand from './components/ui/SuchZustand.svelte';
	import { Search } from '@lucide/svelte';
	import AnliegenWidget from './components/portal/AnliegenWidget.svelte';
	import PortalTrefferkarte from './components/portal/PortalTrefferkarte.svelte';
	import PortalUeberblick from './components/portal/PortalUeberblick.svelte';
	import Reiter from './components/ui/Reiter.svelte';
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

	// Per-book form state
	let reservierungForms = $state(
		/** @type {Record<string, { open: boolean, klasse: string, anzahl: number, notiz: string, loading: boolean, success: string|null, error: string|null, idempotencyKey: string|null }>} */ ({})
	);

	let searchTimeout = /** @type {any} */ (null);

	/**
	 * Offene Reservierungen aller Lehrkräfte (Titel, Klasse, Menge — ohne Personen):
	 * die Warteschlange. Reservieren sperrt nichts; wer denselben Titel reserviert,
	 * stellt sich an. Diese Liste macht das VOR dem Klick sichtbar.
	 * @type {{ titel_id: string, titel: string, klasse: string, anzahl: number, erstellt_am: string }[]}
	 */
	let offeneReservierungen = $state([]);

	async function ladeOffeneReservierungen() {
		try {
			const res = await apiFetch('/api/reservierungen/klassensatz/offen');
			if (res.ok) {
				const data = await res.json();
				// Nur Arrays übernehmen — eine unerwartete Antwort darf die Anzeige
				// nicht mit einem .filter-Absturz aus dem Rendern werfen.
				if (Array.isArray(data)) offeneReservierungen = data;
			}
		} catch {
			/* Anzeige ist Zusatzinfo — ohne sie bleibt das Portal benutzbar */
		}
	}

	$effect(() => {
		ladeOffeneReservierungen();
		ladeAnliegen();
	});

	/** @param {string} titelId */
	function warteschlangeFuer(titelId) {
		return offeneReservierungen.filter((o) => o.titel_id === titelId);
	}

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

	/**
	 * Legt das Formular-Objekt für einen Titel an, falls es fehlt.
	 * Darf NUR aus Event-Handlern/asynchronem Code aufgerufen werden —
	 * eine Zuweisung an $state während des Template-Renderns wirft in
	 * Svelte 5 `state_unsafe_mutation` und bricht das Rendern der
	 * Suchtreffer komplett ab (so konnten Lehrkräfte real nicht suchen).
	 * @param {string} titelId
	 */
	function ensureForm(titelId) {
		if (!reservierungForms[titelId]) {
			reservierungForms[titelId] = {
				open: false,
				klasse: user?.klasse ?? '',
				anzahl: 1,
				notiz: '',
				loading: false,
				success: null,
				error: null,
				idempotencyKey: /** @type {string | null} */ (null)
			};
		}
		return reservierungForms[titelId];
	}

	/**
	 * Reine Lese-Sicht fürs Template — mutiert nie.
	 * @param {string} titelId
	 */
	function getForm(titelId) {
		return (
			reservierungForms[titelId] ?? {
				open: false,
				klasse: user?.klasse ?? '',
				anzahl: 1,
				notiz: '',
				loading: false,
				success: null,
				error: null,
				idempotencyKey: /** @type {string | null} */ (null)
			}
		);
	}

	/**
	 * @param {string} titelId
	 */
	function toggleForm(titelId) {
		const f = ensureForm(titelId);
		f.open = !f.open;
		f.success = null;
		f.error = null;
	}

	/**
	 * @param {string} titelId
	 */
	async function submitReservierung(titelId) {
		const f = ensureForm(titelId);
		if (f.loading) return; // Doppelklick abfangen, bevor die Anfrage überhaupt rausgeht
		if (!f.klasse.trim()) {
			f.error = 'Bitte Klasse angeben.';
			return;
		}
		f.loading = true;
		f.error = null;
		f.success = null;
		// Idempotenz-Schlüssel pro Absende-Vorgang: Überholt ein Doppelklick den loading-
		// Guard (oder klemmt das Netz und der Client wiederholt), geht DERSELBE Schlüssel
		// raus — der Server macht daraus ein No-op statt einer zweiten Reservierung/Mail.
		if (!f.idempotencyKey) f.idempotencyKey = crypto.randomUUID();
		try {
			const res = await apiFetch('/api/reservierungen/klassensatz', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					titel_id: titelId,
					klasse: f.klasse,
					anzahl: f.anzahl,
					notiz: f.notiz,
					idempotency_key: f.idempotencyKey
				})
			});
			if (res.ok) {
				f.idempotencyKey = null; // erfolgreich → der nächste Vorgang bekommt einen neuen
				const vorher = warteschlangeFuer(titelId);
				f.success =
					vorher.length > 0
						? `Reservierungsanfrage gesendet — dein Satz ist nach ${vorher.map((o) => o.klasse).join(', ')} an der Reihe.`
						: 'Reservierungsanfrage wurde gesendet!';
				f.open = false;
				ladeOffeneReservierungen();
			} else {
				// Die Antwort ist apierrors-JSON ({"error": "nur 2 Exemplare im Bestand …"}) —
				// roh angezeigt las die Lehrkraft Klammern statt der Meldung.
				const txt = await res.text();
				let meldung = txt;
				try {
					meldung = JSON.parse(txt).error || txt;
				} catch {
					/* Rohtext behalten */
				}
				f.error = meldung || 'Fehler beim Senden.';
			}
		} catch (e) {
			f.error = String(e);
		} finally {
			f.loading = false;
		}
	}
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
			{ id: 'buecher', label: 'Bücher & Klassensätze' },
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
						form={getForm(titelId)}
						warteschlange={warteschlangeFuer(titelId)}
						ontoggle={() => toggleForm(titelId)}
						onsenden={() => submitReservierung(titelId)}
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
			<PortalUeberblick
				reservierungen={offeneReservierungen}
				anliegen={eigeneAnliegen}
				onanliegen={() => (reiter = 'anliegen')}
			/>
		{/if}
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
			class="shrink-0 w-4 h-4 border-2 border-blue-500/40 border-t-blue-500 rounded-full animate-spin"
			aria-hidden="true"
		></div>
	{/if}
{/snippet}
