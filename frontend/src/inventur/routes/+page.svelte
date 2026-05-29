<!--
	+page.svelte (Startseite)

	Orchestriert die Gast-Ansicht der Inventur-App:
	Login, Navigation, Filterung und Anzeige der Bücher/Klassen.
	Die eigentliche Logik und UI-Blöcke sind in Unterkomponenten ausgelagert.
-->
<script>
	import { fade } from "svelte/transition";
	import { onMount } from "svelte";
	import { appState } from "$lib/store.svelte.js";
	import GuestLoginView from "$lib/components/GuestLoginView.svelte";
	import BuchRasterStartseite from "$lib/components/BuchRasterStartseite.svelte";
	import KlassenUebersichtStartseite from "$lib/components/KlassenUebersichtStartseite.svelte";
	import StartseitenFilter from "$lib/components/StartseitenFilter.svelte";
	import { holeAuthStatus } from "$lib/auth_api.js";
	import {
		gastLoginAusfuehren,
		buecherLaden,
		echteKlassenLaden,
		buecherNachKlassenGruppieren,
		bestandsFarbe,
	} from "$lib/startseiten_api.js";

	// --- Zustandsvariablen ---
	let viewMode = $state("suche");
	let searchQuery = $state("");
	let selectedZweig = $state("");
	let selectedJahrgang = $state("");
	let klasseSearchQuery = $state("");
	let selectedKlasse = $state("");
	let isKlasseDropdownOpen = $state(false);

	/** @type {any[]} */
	let books = $state([]);
	/** @type {any[]} */
	let realClasses = $state([]);
	let guestPassword = $state("");
	let loginError = $state("");
	let loading = $state(false);

	// --- Abgeleitete Werte ---
	let classes = $derived(buecherNachKlassenGruppieren(books));
	let klassenList = $derived(
		realClasses.map((c) => c.name.replace("Klasse ", "")),
	);

	// --- WZ-Synonyme für Suchbegriffe auf der Startseite ---
	/** @type {Record<string, string>} */
	const suchSynonyme = {
		powi: "politik",
		mathe: "mathematik",
		eng: "englisch",
		deu: "deutsch",
		franz: "französisch",
		bio: "biologie",
		che: "chemie",
		phy: "physik",
		geo: "geographie",
		info: "informatik",
		lat: "latein",
		span: "spanisch",
		rel: "religion",
		reli: "religion",
	};

	let filteredBooks = $derived(
		(Array.isArray(books) ? books : []).filter((/** @type {any} */ b) => {
			let q = searchQuery.toLowerCase().trim();
			if (q in suchSynonyme) {
				q = suchSynonyme[q];
			}
			return (
				q === "" ||
				(b.title && b.title.toLowerCase().includes(q)) ||
				(b.isbn && b.isbn.toLowerCase().includes(q)) ||
				(b.author && b.author.toLowerCase().includes(q)) ||
				(b.subject && b.subject.toLowerCase().includes(q))
			);
		}),
	);

	let filteredClasses = $derived(
		(Array.isArray(classes) ? classes : []).filter((cls) => {
			const zw =
				selectedZweig === "" ||
				cls.books.some((/** @type {any} */ b) => b.track === selectedZweig);
			const jg =
				selectedJahrgang === "" ||
				cls.name.includes(`Klasse ${selectedJahrgang}`);
			return zw && jg;
		}),
	);

	let filteredRealClasses = $derived(
		(Array.isArray(realClasses) ? realClasses : []).filter(
			(cls) => selectedKlasse === "" || cls.name === selectedKlasse,
		),
	);

	let filteredKlassenList = $derived(
		klassenList.filter((k) =>
			k.toLowerCase().includes(klasseSearchQuery.toLowerCase()),
		),
	);

	let displayLimit = $state(50);
	let paginatedBooks = $derived(filteredBooks.slice(0, displayLimit));

	$effect(() => {
		filteredBooks;
		displayLimit = 50;
	});

	// --- Initialisierung ---
	onMount(() => {
		initialisiereSeite();
	});

	async function initialisiereSeite() {
		loading = true;
		try {
			const status = await holeAuthStatus();
			if (!status.authenticated) {
				appState.guestAuthenticated = false;
				return;
			}
			await ladeDaten();
		} finally {
			loading = false;
		}
	}

	async function ladeDaten() {
		try {
			books = await buecherLaden();
			realClasses = await echteKlassenLaden();
			appState.guestAuthenticated = true;
		} catch {
			appState.guestAuthenticated = false;
		}
	}

	// Automatischer Reset der Auswahl, wenn das Suchfeld geleert wird
	$effect(() => {
		if (klasseSearchQuery === "" && selectedKlasse !== "") {
			selectedKlasse = "";
		}
	});

	// Wenn exakt eine Klasse getippt wurde, die existiert, diese auch selektieren
	$effect(() => {
		if (klasseSearchQuery !== "" && selectedKlasse === "") {
			const exactMatch = realClasses.find(
				(c) =>
					c.name === klasseSearchQuery ||
					c.name === `Klasse ${klasseSearchQuery}`,
			);
			if (exactMatch) {
				selectedKlasse = exactMatch.name;
			}
		}
	});

	async function performGuestLogin() {
		loginError = "";
		loading = true;
		try {
			await gastLoginAusfuehren(guestPassword);
			appState.guestAuthenticated = true;
			ladeDaten();
		} catch (e) {
			const error = /** @type {any} */ (e);
			loginError = error.message || String(error);
		} finally {
			loading = false;
		}
	}

	/**
	 * @param {string} klasse
	 */
	function selectKlasse(klasse) {
		selectedKlasse = klasse;
		klasseSearchQuery = klasse;
		isKlasseDropdownOpen = false;
	}
</script>

<div class="w-full text-zinc-100 font-sans">
	<div class="w-full transition-all duration-300">
		{#if !appState.guestAuthenticated}
			<GuestLoginView
				{loginError}
				{loading}
				onLogin={performGuestLogin}
			/>
		{:else}
			<StartseitenFilter
				bind:viewMode
				bind:searchQuery
				bind:selectedZweig
				bind:selectedJahrgang
				bind:klasseSearchQuery
				bind:isKlasseDropdownOpen
				{filteredKlassenList}
				onSelectKlasse={selectKlasse}
			/>

			<main class="relative">
				{#if viewMode === "suche"}
					<div
						in:fade={{ duration: 300, delay: 150 }}
						out:fade={{ duration: 150 }}
						role="tabpanel"
						id="content-suche"
						aria-labelledby="tab-suche"
					>
						<BuchRasterStartseite
							filteredBooks={paginatedBooks}
						/>
						{#if displayLimit < filteredBooks.length}
							<div class="mt-8 flex justify-center">
								<button
									class="px-6 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-100 font-semibold rounded-full border border-zinc-700/60 shadow-lg transition-all cursor-pointer"
									onclick={() => (displayLimit += 50)}
								>
									Mehr laden ({filteredBooks.length -
										displayLimit} weitere)
								</button>
							</div>
						{/if}
					</div>
				{:else if viewMode === "jahrgaenge"}
					<div
						in:fade={{ duration: 300, delay: 150 }}
						out:fade={{ duration: 150 }}
						class="space-y-8"
						role="tabpanel"
						id="content-jahrgaenge"
						aria-labelledby="tab-jahrgaenge"
					>
						<KlassenUebersichtStartseite
							{filteredClasses}
							getStockColor={bestandsFarbe}
						/>
					</div>
				{:else}
					<div
						in:fade={{ duration: 300, delay: 150 }}
						out:fade={{ duration: 150 }}
						class="space-y-8"
						role="tabpanel"
						id="content-schulklassen"
						aria-labelledby="tab-schulklassen"
					>
						{#key selectedKlasse}
							<KlassenUebersichtStartseite
								filteredClasses={filteredRealClasses}
								getStockColor={bestandsFarbe}
							/>
						{/key}
					</div>
				{/if}
			</main>
		{/if}
	</div>
</div>
