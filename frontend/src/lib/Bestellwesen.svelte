<script>
	import { onMount } from 'svelte';
	import { ChevronLeft } from '@lucide/svelte';
	import { druckWarteschlange } from './stores/druckWarteschlange.svelte.js';
	import { bestellStore } from './stores/bestellStore.svelte.js';
	import { uiStore } from './stores/uiStore.svelte.js';
	import { snackbars } from './stores/snackbars.svelte.js';
	import Seitenrahmen from './components/layout/Seitenrahmen.svelte';

	import BestellEntwurf from './components/bestellungen/BestellEntwurf.svelte';
	import Wareneingang from './components/bestellungen/Wareneingang.svelte';
	import Bestellbedarf from './components/bestellungen/Bestellbedarf.svelte';
	import BestellHistorie from './components/bestellungen/BestellHistorie.svelte';
	import BestelllinkHinweis from './components/bestellungen/BestelllinkHinweis.svelte';
	import Reiter from './components/ui/Reiter.svelte';
	import KlassensatzReservierungen from './components/bestellungen/KlassensatzReservierungen.svelte';
	import AnliegenListe from './components/bestellungen/AnliegenListe.svelte';
	import { resthoehe } from './actions/resthoehe.js';

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

	// Bestellspalte ein-/ausklappbar. Sie belegt ein Drittel der Breite, auch wenn der
	// Warenkorb leer ist — währenddaneben Titel wie „LMF-Bigalke/Köhler: Mathematik —
	// Hessen — Ausgabe 2016/Grundkurs 2. Halbjahr — Band Q2" abgeschnitten werden.
	//
	// Bewusst OFFEN als Vorgabe: Wer die Seite kennt, soll sie unverändert vorfinden, und
	// das E2E-Gate (bestellung-erreichbar.spec.js) prüft genau diesen Zustand.
	// Bewusst NICHT gespeichert: Ein Wert für die Fensteraufteilung gehört weder in den
	// localStorage (mehrere Arbeitsplätze) noch als geteilter Zustand ins Backend — an
	// zwei Bildschirmen mit verschiedenen Grössen wäre „meine Breite" die falsche Vorgabe.
	// Der Zustand gilt deshalb für die Sitzung.
	let railOffen = $state(true);

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
			{ id: 'anliegen', label: 'Meldungen' }
		]}
	/>

	{#if aktiverReiter === 'bestellungen'}
		{#key 'bestellungen'}
			<div class="flex flex-col gap-5">
				<!-- Zwei Bereiche statt zwei Karten. Die Trennung traegt eine senkrechte
				     Haarlinie an der Bestellspalte (unten), nicht ein Rahmen je Block —
				     dasselbe Muster wie auf der Signaturen-Seite. Waagerechter Abstand
				     deshalb 0: Der Abstand entsteht aus der Polsterung links und rechts
				     der Linie, sonst stuenden 24 px Luecke UND 24 px Polsterung. -->
				<div class="grid grid-cols-1 gap-y-6 lg:grid-cols-12 lg:gap-x-0 items-start">
					<!-- HERO: der Bestellbedarf ist die tägliche Arbeitsfläche.
					     Die Spaltenbreiten stehen je Zustand VOLLSTÄNDIG da (lg und xl), statt sich
					     auf die Reihenfolge im class-Attribut zu verlassen: Bei Utilities gleicher
					     Spezifität entscheidet die Reihenfolge im Stylesheet, nicht die im Markup. -->
					<div
						class="min-w-0 lg:pr-6 {railOffen
							? 'lg:col-span-7 xl:col-span-8'
							: 'lg:col-span-11 xl:col-span-11'}"
					>
						<Bestellbedarf
							vorschlaege={bestellStore.recommendations}
							onAddToCart={(book) => bestellStore.addToCart(book)}
						/>
					</div>

					<!-- Die Bestellspalte klebt beim Scrollen und reicht bis zum unteren Rand der Seite
					     (actions/resthoehe.js): Ein voller Warenkorb scrollt in ihr, Absenden bleibt
					     erreichbar, und die Trefferliste der Titelsuche läge bei leerem Korb sonst unter
					     dem Rand (Gate: e2e/bestellung-erreichbar.spec.js). Oben und rechts reicht sie
					     8 px über den Inhalt, so weit wie die Trefferfläche der Symbolknöpfe am Rand. -->
					<div
						use:resthoehe={{ haftet: true }}
						id="bestellspalte"
						class="space-y-3 lg:sticky lg:top-0 lg:-mt-2 lg:-mr-2 lg:pt-2 lg:pr-2 lg:h-(--resthoehe) lg:overflow-y-auto {railOffen
							? 'lg:col-span-5 xl:col-span-4 lg:border-l lg:border-outline-variant lg:pl-6'
							: 'lg:col-span-1 xl:col-span-1'}"
					>
						{#if !railOffen}
							<!-- Eingeklappt: ein beschrifteter Streifen, kein stummes Rechteck. Die Zahl
							     im Korb bleibt sichtbar — sonst müsste man aufklappen, um zu sehen, ob
							     überhaupt etwas drin ist. -->
							<button
								onclick={() => (railOffen = true)}
								aria-expanded="false"
								aria-controls="bestellpanel"
								class="hidden lg:flex w-full flex-col items-center gap-3 py-4 px-2 rounded-xl border border-outline-variant bg-surface-container-lowest cursor-pointer"
							>
								<ChevronLeft class="h-4 w-4 text-on-surface-variant" aria-hidden="true" />
								<span
									class="text-xs font-bold text-on-surface-variant tracking-wide"
									style="writing-mode: vertical-rl">Deine Bestellung</span
								>
								{#if bestellStore.totalQty > 0}
									<span
										class="min-w-6 h-6 flex items-center justify-center rounded-full bg-primary text-on-primary text-label-small font-bold px-1.5 tabular-nums"
										>{bestellStore.totalQty}</span
									>
								{/if}
							</button>
						{/if}

						<!-- Das Panel bleibt eingehängt und wird nur verborgen: Beim Aus- und
						     Wiedereinhängen gingen Lieferantenwahl und Suchfeld verloren.
						     max-lg:block, weil unterhalb von lg gar nicht nebeneinander gelegt wird —
						     ohne das wäre die Bestellung auf dem Tablet verschwunden, nachdem jemand
						     am grossen Bildschirm eingeklappt hat. -->
						<div id="bestellpanel" class={railOffen ? '' : 'hidden max-lg:block'}>
							<BestellEntwurf onCollapse={() => (railOffen = false)} />
						</div>
					</div>
				</div>
			</div>
		{/key}
	{/if}

	{#if aktiverReiter === 'wareneingang'}
		<!-- Eigener Reiter statt Unteransicht hinter einem Banner. Vorher fuehrte der einzige
		     Weg hierher ueber einen Streifen ueber dem Bestellbedarf; wer ihn wegdachte, fand
		     den Wareneingang nicht mehr. Als Reiter steht er gleichrangig neben den anderen
		     Ansichten dieser Seite — und die Zahl im Badge sagt dasselbe wie der Streifen,
		     ohne die halbe Breite zu belegen. -->
		<Wareneingang
			incomingShipments={bestellStore.incomingShipments}
			onBack={() => (aktiverReiter = 'bestellungen')}
			onReceived={beiWareneingang}
		/>
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
