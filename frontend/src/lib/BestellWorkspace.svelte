<script>
	import { onMount } from 'svelte';
	import { ChevronLeft } from '@lucide/svelte';
	import { printQueue } from './stores/printQueue.svelte.js';
	import { orderStore } from './stores/orderStore.svelte.js';
	import { uiStore } from './stores/uiStore.svelte.js';
	import { toastStore } from './stores/toastStore.svelte.js';
	import PageShell from './components/layout/PageShell.svelte';

	import OrderCreationPanel from './components/bestellungen/OrderCreationPanel.svelte';
	import WareneingangView from './components/bestellungen/WareneingangView.svelte';
	import OrderRecommendations from './components/bestellungen/OrderRecommendations.svelte';
	import BestellHistorie from './components/bestellungen/BestellHistorie.svelte';
	import BestellBerichte from './components/bestellungen/BestellBerichte.svelte';
	import BestelllinkHinweis from './components/bestellungen/BestelllinkHinweis.svelte';
	import KlassensatzReservierungen from './components/bestellungen/KlassensatzReservierungen.svelte';
	import AnliegenListe from './components/bestellungen/AnliegenListe.svelte';

	let activeTab = $state('bestellungen');

	/**
	 * Exemplare im Zulauf — speist das Badge am Wareneingang-Reiter. Bis zum 09.08.2026
	 * stand dieselbe Zahl als Streifen ueber dem Bestellbedarf; M3 kennt dafuer das Badge
	 * am Ziel, die Banner-Komponente wurde mit M2 gestrichen.
	 */
	const zulaufExemplare = $derived(
		orderStore.incomingShipments.reduce(
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
		orderStore.init();
		// Zusätzlich zu init(): Das Laden dort ist 60 Sekunden lang gecacht. Wer die
		// öffentliche Adresse gerade in den Einstellungen nachgetragen hat und zurückkommt,
		// stünde sonst vor einem Hinweis, der bereits erledigt ist — und würde ihn beim
		// nächsten Mal nicht mehr ernst nehmen.
		orderStore.loadKonfiguration();
	});

	/** @param {any[]} receivedItems */
	async function handleShipmentReceived(receivedItems) {
		activeTab = 'bestellungen';
		await orderStore.loadIncomingShipments();
		orderStore.loadRecommendations();
		// Zaehlt das Badge am Druck-Center neu: Frisch eingebuchte Exemplare tragen noch
		// kein Etikett, die Arbeit ist also gerade dorthin gewandert.
		uiStore.fetchOffeneEtiketten();

		// Der stehende Streifen ist weg, die ÜBERGABE aber nicht: Wer gerade eingebucht hat,
		// soll genau DIESE Exemplare drucken koennen — nicht sie zwischen tausenden offenen
		// Etiketten im Druck-Center wiederfinden muessen. Deshalb eine Snackbar mit Aktion
		// (M3: genau eine Folgehandlung), die printQueue.copies wie zuvor befuellt.
		//
		// Der Unterschied zum Streifen ist die Lebensdauer: Der Hinweis beschreibt einen
		// MOMENT. Der dauerhafte Zustand („es liegen Etiketten an") steht im Badge am
		// Druck-Center und nicht mehr auf dieser Seite.
		const ohneEtikett = receivedItems.filter((item) => !item.etikett_gedruckt);
		if (ohneEtikett.length > 0) {
			toastStore.addToast(
				`Eingebucht. ${ohneEtikett.length} ${ohneEtikett.length === 1 ? 'Exemplar braucht' : 'Exemplare brauchen'} noch ein Etikett.`,
				'success',
				{
					label: 'Etiketten drucken',
					onClick: () => {
						printQueue.copies = ohneEtikett;
					}
				}
			);
		} else {
			toastStore.addToast('Eingebucht.', 'success');
		}
	}

	/**
	 * Höhe der Bestellspalte, damit sie nie unter den Fensterrand wächst.
	 *
	 * Aus dem Betrieb gemeldet als „Bestellung abgeschnitten": Die Spalte klebt beim
	 * Scrollen (sticky). Füllt sich der Warenkorb, wird sie höher als das Fenster — bei
	 * 1366×700 gemessen 1427 px, also 979 px unter dem Rand. Der Absenden-Knopf war dann
	 * nur noch zu erreichen, indem man die 105 Titel der Bedarfsliste daneben
	 * durchscrollte, bis der klebende Rahmen endete.
	 *
	 * Warum gemessen statt einer festen CSS-Grenze: Ein calc(100vh - fester Abzug) kann
	 * nicht stimmen, weil der Anfang der Spalte wandert — mit eingeblendetem
	 * Backup-Banner beginnt sie rund 80 px tiefer, und beim Scrollen wandert sie nach
	 * oben, bis sie klebt. Ein Versuch mit 8rem Abzug liess immer noch 125 px überstehen.
	 * Deshalb rechnet die Spalte mit ihrem tatsächlichen Abstand zum oberen Rand.
	 */
	/** @type {HTMLElement|undefined} */
	let rail = $state();
	let railMaxHeight = $state('');

	$effect(() => {
		if (!rail) return;
		const messen = () => {
			if (!rail) return;
			const oben = rail.getBoundingClientRect().top;
			// 8 px Luft unten, damit die Spalte nicht am Rand klebt.
			railMaxHeight = `${Math.max(240, Math.round(window.innerHeight - oben - 8))}px`;
		};
		messen();
		// Beim Scrollen wandert der Anfang, bis die Spalte klebt — dann wird mehr Platz frei.
		window.addEventListener('scroll', messen, true);
		window.addEventListener('resize', messen);

		// Und wenn sich das Layout ÜBER der Spalte ändert. Ohne das rechnete der Effekt nur
		// einmal beim Einhängen — zu einem Zeitpunkt, an dem der Backup-Hinweis und der
		// Wareneingangs-Streifen noch nicht standen. Die Spalte begann danach rund 100 px
		// tiefer, behielt aber die zu grosszügige Höhe und ragte wieder unter den Rand.
		const beobachter = new ResizeObserver(messen);
		beobachter.observe(document.body);
		beobachter.observe(rail);

		return () => {
			window.removeEventListener('scroll', messen, true);
			window.removeEventListener('resize', messen);
			beobachter.disconnect();
		};
	});
</script>

<PageShell>
	<!-- Über den Reitern und nicht in einem von ihnen: Die fehlende Adresse betrifft das
	     Bestellen (Mail ohne Link) UND die Historie (Bestätigung, die nie kommt). -->
	<BestelllinkHinweis />

	<!-- Reiterzeile: reine Navigation, keine Aktionen. Genau deshalb steht „Nachdrucken"
	     NICHT hier — in M3 wechseln Reiter gleichrangige Ansichten EINES Ziels, sie fuehren
	     keine Aktionen aus und springen nicht zu anderen Zielen. Das Nachdrucken liegt im
	     Druck-Center und traegt sein Badge dort (Sidebar.svelte).

	     role=tablist/tab und aria-selected: Vorher waren das nackte <button>, ein
	     Screenreader hoerte fuenf zusammenhanglose Knoepfe statt einer Reitergruppe. -->
	<div
		role="tablist"
		aria-label="Bereiche des Bestellwesens"
		class="flex items-end gap-6 border-b border-slate-200 shrink-0 overflow-x-auto"
	>
		{#snippet tab(id, label, anzahl = 0)}
			<button
				role="tab"
				aria-selected={activeTab === id}
				onclick={() => (activeTab = id)}
				class="flex shrink-0 items-center gap-2 pb-3 text-sm font-semibold whitespace-nowrap border-b-2 transition-colors cursor-pointer {activeTab ===
				id
					? 'border-blue-600 text-blue-700'
					: 'border-transparent text-slate-500 hover:text-slate-800'}"
			>
				{label}
				{#if anzahl > 0}
					<span
						class="min-w-5 h-5 flex items-center justify-center rounded-full bg-error text-on-error text-label-small font-bold px-1 tabular-nums"
						aria-label="{anzahl} offen">{anzahl > 999 ? '999+' : anzahl}</span
					>
				{/if}
			</button>
		{/snippet}
		{@render tab('bestellungen', 'Bestellungen')}
		{@render tab('wareneingang', 'Wareneingang', zulaufExemplare)}
		{@render tab('historie', 'Bestellhistorie')}
		{@render tab('berichte', 'Berichte')}
		{@render tab('klassensaetze', 'Klassensatz-Reservierungen', uiStore.pendingReservierungen)}
		{@render tab('anliegen', 'Wünsche & Meldungen')}
	</div>

	{#if activeTab === 'bestellungen'}
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
						<OrderRecommendations
							recommendations={orderStore.recommendations}
							onAddToCart={(book) => orderStore.addToCart(book)}
						/>
					</div>

					<!-- RAIL: die Bestellung wächst sichtbar mit, bleibt beim Scrollen stehen. Der
					     Etiketten-Hinweis ist raus (steht jetzt oben bei den offenen Aufgaben) — die
					     Spalte trägt damit nur noch EINE Sache: die Bestellung, die man gerade schreibt.
					     Aus dem Betrieb gemeldet als „Bestellung abgeschnitten". Gemessen bei
					     1366×700 mit fünf Positionen im Korb: Die Spalte wurde 1288 px hoch und
					     ragte 826 px unter den Fensterrand — der Absenden-Knopf war nur noch zu
					     erreichen, indem man die ganze Bestellbedarfs-Liste daneben (105 Titel)
					     nach unten scrollte, bis der klebende Rahmen endete. Erreichbar im
					     technischen Sinn, unbrauchbar im täglichen.
					     Höhengrenze plus eigener Scrollbalken: Wird die Spalte zu hoch, scrollt sie
					     in sich selbst. Der Warenkorb bleibt damit dort bedienbar, wo er steht.
					     Verankert im E2E-Gate e2e/bestellung-erreichbar.spec.js. -->
					<div
						bind:this={rail}
						id="bestellspalte"
						style:--rail-max={railMaxHeight}
						class="space-y-3 lg:sticky lg:top-2 lg:max-h-(--rail-max) lg:overflow-y-auto {railOffen
							? 'lg:col-span-5 xl:col-span-4 lg:border-l lg:border-slate-200 lg:pl-6'
							: 'lg:col-span-1 xl:col-span-1'}"
					>
						{#if !railOffen}
							<!-- Eingeklappt: ein beschrifteter Streifen, kein stummes Rechteck. Die Zahl
							     im Korb bleibt sichtbar — sonst müsste man aufklappen, um zu sehen, ob
							     überhaupt etwas drin ist. -->
							<button
								onclick={() => (railOffen = true)}
								aria-label="Bestellspalte ausklappen"
								aria-expanded="false"
								aria-controls="bestellpanel"
								class="hidden lg:flex w-full flex-col items-center gap-3 py-4 px-2 rounded-xl border border-slate-200 bg-white shadow-sm hover:border-blue-400 hover:bg-blue-50/40 transition-colors cursor-pointer"
							>
								<ChevronLeft class="h-4 w-4 text-slate-400" aria-hidden="true" />
								<span
									class="text-xs font-bold text-slate-600 tracking-wide"
									style="writing-mode: vertical-rl">Deine Bestellung</span
								>
								{#if orderStore.totalQty > 0}
									<span
										class="min-w-6 h-6 flex items-center justify-center rounded-full bg-blue-600 text-white text-label-small font-bold px-1.5 tabular-nums"
										>{orderStore.totalQty}</span
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
							<OrderCreationPanel onCollapse={() => (railOffen = false)} />
						</div>
					</div>
				</div>
			</div>
		{/key}
	{/if}

	{#if activeTab === 'wareneingang'}
		<!-- Eigener Reiter statt Unteransicht hinter einem Banner. Vorher fuehrte der einzige
		     Weg hierher ueber einen Streifen ueber dem Bestellbedarf; wer ihn wegdachte, fand
		     den Wareneingang nicht mehr. Als Reiter steht er gleichrangig neben den anderen
		     Ansichten dieser Seite — und die Zahl im Badge sagt dasselbe wie der Streifen,
		     ohne die halbe Breite zu belegen. -->
		<WareneingangView
			incomingShipments={orderStore.incomingShipments}
			onBack={() => (activeTab = 'bestellungen')}
			onReceived={handleShipmentReceived}
		/>
	{/if}

	{#if activeTab === 'historie'}
		<BestellHistorie />
	{/if}

	{#if activeTab === 'berichte'}
		<BestellBerichte suppliers={orderStore.suppliers} />
	{/if}

	{#if activeTab === 'klassensaetze'}
		<KlassensatzReservierungen />
	{/if}

	{#if activeTab === 'anliegen'}
		<AnliegenListe />
	{/if}
</PageShell>
