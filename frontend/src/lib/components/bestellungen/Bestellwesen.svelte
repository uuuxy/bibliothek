<script>
	import { onMount } from 'svelte';
	import { druckWarteschlange } from '../../stores/druckWarteschlange.svelte.js';
	import { bestellStore } from '../../stores/bestellStore.svelte.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import Seitenrahmen from '../layout/Seitenrahmen.svelte';

	import BestellEntwurf from './BestellEntwurf.svelte';
	import BestellZusammenfassung from './BestellZusammenfassung.svelte';
	import Wareneingang from './Wareneingang.svelte';
	import BestellHistorie from './BestellHistorie.svelte';
	import BestelllinkHinweis from './BestelllinkHinweis.svelte';
	import Reiter from '../ui/Reiter.svelte';
	import KlassensatzReservierungen from './KlassensatzReservierungen.svelte';
	import AnliegenListe from './AnliegenListe.svelte';
	import { resthoehe } from '../../actions/resthoehe.js';

	let aktiverReiter = $state('bestellungen');

	/**
	 * Exemplare im Zulauf; speist das Badge am Wareneingang-Reiter. M3 zeigt eine solche Zahl
	 * als Badge am Ziel, ein Banner kennt es nicht mehr.
	 */
	const zulaufExemplare = $derived(
		bestellStore.incomingShipments.reduce(
			(/** @type {number} */ summe, /** @type {any} */ lieferung) =>
				summe +
				lieferung.items.reduce((/** @type {number} */ s, /** @type {any} */ i) => s + i.menge, 0),
			0
		)
	);

	onMount(() => {
		bestellStore.init();
		// Zusätzlich zu init(): Das Laden dort ist 60 Sekunden lang gecacht. Wer die
		// öffentliche Adresse gerade in den Einstellungen nachgetragen hat und zurückkommt,
		// stünde sonst vor einem Hinweis, der bereits erledigt ist — und würde ihn beim
		// nächsten Mal nicht mehr ernst nehmen.
		bestellStore.loadKonfiguration();
	});

	/** @param {any[]} receivedItems */
	async function beiWareneingang(receivedItems) {
		aktiverReiter = 'bestellungen';
		await bestellStore.loadIncomingShipments();
		bestellStore.loadRecommendations();
		// Zaehlt das Badge am Druck-Center neu: Frisch eingebuchte Exemplare tragen noch
		// kein Etikett, die Arbeit ist also gerade dorthin gewandert.
		uiStore.fetchOffeneEtiketten();

		// Der stehende Streifen ist weg, die ÜBERGABE aber nicht: Wer gerade eingebucht hat,
		// soll genau DIESE Exemplare drucken koennen — nicht sie zwischen tausenden offenen
		// Etiketten im Druck-Center wiederfinden muessen. Deshalb eine Snackbar mit Aktion
		// (M3: genau eine Folgehandlung), die druckWarteschlange.copies wie zuvor befuellt.
		//
		// Der Unterschied zum Streifen ist die Lebensdauer: Der Hinweis beschreibt einen
		// MOMENT. Der dauerhafte Zustand („es liegen Etiketten an") steht im Badge am
		// Druck-Center und nicht mehr auf dieser Seite.
		const ohneEtikett = receivedItems.filter((item) => !item.etikett_gedruckt);
		if (ohneEtikett.length > 0) {
			snackbars.addToast(
				`Eingebucht. ${ohneEtikett.length} ${ohneEtikett.length === 1 ? 'Exemplar braucht' : 'Exemplare brauchen'} noch ein Etikett.`,
				'success',
				{
					label: 'Etiketten drucken',
					onClick: () => {
						druckWarteschlange.copies = ohneEtikett;
					}
				}
			);
		} else {
			snackbars.addToast('Eingebucht.', 'success');
		}
	}
</script>

<Seitenrahmen>
	<!-- Über den Reitern und nicht in einem von ihnen: Die fehlende Adresse betrifft das
	     Bestellen (Mail ohne Link) UND die Historie (Bestätigung, die nie kommt). -->
	<BestelllinkHinweis />

	<!-- Reiterzeile: reine Navigation, keine Aktionen. Deshalb steht „Nachdrucken" nicht hier:
	     In M3 wechseln Reiter gleichrangige Ansichten eines Ziels, sie führen keine Aktionen
	     aus und springen nicht zu anderen Zielen. Das Nachdrucken liegt im Druck-Center und
	     trägt sein Badge dort (Seitenleiste.svelte). -->
	<Reiter
		etikett="Bereiche des Bestellwesens"
		aktiv={aktiverReiter}
		onwahl={(id) => (aktiverReiter = id)}
		klasse="shrink-0"
		reiter={[
			{ id: 'bestellungen', label: 'Bestellungen' },
			{ id: 'wareneingang', label: 'Wareneingang', anzahl: zulaufExemplare },
			{ id: 'historie', label: 'Bestellhistorie' },
			{
				id: 'klassensaetze',
				label: 'Klassensatz-Reservierungen',
				anzahl: uiStore.pendingReservierungen
			},
			// Dieselbe Zahl, die die Seitenleiste an „Bestellungen“ mitzählt (wartendeArbeit).
			{ id: 'anliegen', label: 'Meldungen', anzahl: uiStore.offeneAnliegen }
		]}
	/>

	{#if aktiverReiter === 'bestellungen'}
		{#key 'bestellungen'}
			<!-- Aufteilung wie im Warenkorb des Google Store: breit die Bestellung mit dem
			     Bestellbedarf darunter, daneben die Zusammenfassung. M3 Canonical layouts,
			     Supporting pane: der Hauptbereich „occupies the majority of the window (typically
			     about two-thirds)", der Nebenbereich daneben „Fixed (360 dp)"; ab xl 412 dp, die
			     zweite Breite, die M3 für feste Bereiche nennt. Getrennt durch die senkrechte Linie
			     wie auf der Signaturen-Seite; der Abstand entsteht aus der Polsterung beiderseits. -->
			<div class="flex flex-col gap-y-6 lg:flex-row lg:items-start">
				<div class="min-w-0 lg:flex-1 lg:pr-6">
					<BestellEntwurf />
				</div>

				<!-- Die Zusammenfassung klebt beim Scrollen und reicht bis zum unteren Rand der Seite
				     (actions/resthoehe.js): Summe und Knopf bleiben stehen, wie lang die Liste auch
				     ist, und in einem kurzen Fenster scrollt sie in sich (Gate:
				     e2e/bestellung-erreichbar.spec.js). Oben und rechts reicht sie 8 px über den
				     Inhalt, damit overflow-y-auto Fokusring und Trefferflächen am Rand nicht
				     abschneidet. -->
				<aside
					use:resthoehe={{ haftet: true }}
					id="zusammenfassung"
					aria-label="Zusammenfassung der Bestellung"
					class="lg:sticky lg:top-0 lg:-mt-2 lg:-mr-2 lg:h-(--resthoehe) lg:w-90 lg:shrink-0 lg:overflow-y-auto lg:border-l lg:border-outline-variant lg:pt-2 lg:pr-2 lg:pl-6 xl:w-103"
				>
					<BestellZusammenfassung />
				</aside>
			</div>
		{/key}
	{/if}

	{#if aktiverReiter === 'wareneingang'}
		<!-- Eigener Reiter statt Unteransicht hinter einem Banner. Vorher fuehrte der einzige
		     Weg hierher ueber einen Streifen ueber dem Bestellbedarf; wer ihn wegdachte, fand
		     den Wareneingang nicht mehr. Als Reiter steht er gleichrangig neben den anderen
		     Ansichten dieser Seite — und die Zahl im Badge sagt dasselbe wie der Streifen,
		     ohne die halbe Breite zu belegen. -->
		<Wareneingang incomingShipments={bestellStore.incomingShipments} onReceived={beiWareneingang} />
	{/if}

	{#if aktiverReiter === 'historie'}
		<BestellHistorie />
	{/if}

	{#if aktiverReiter === 'klassensaetze'}
		<KlassensatzReservierungen />
	{/if}

	{#if aktiverReiter === 'anliegen'}
		<AnliegenListe />
	{/if}
</Seitenrahmen>
