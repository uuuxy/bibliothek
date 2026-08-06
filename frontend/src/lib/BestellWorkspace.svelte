<script>
	import { onMount } from 'svelte';
	import { ChevronLeft } from '@lucide/svelte';
	import { apiGet } from './apiFetch.js';
	import { orderStore } from './stores/orderStore.svelte.js';
	import { printQueue } from './stores/printQueue.svelte.js';
	import { uiStore } from './stores/uiStore.svelte.js';

	import OrderCreationPanel from './components/bestellungen/OrderCreationPanel.svelte';
	import IncomingShipments from './components/bestellungen/IncomingShipments.svelte';
	import WareneingangView from './components/bestellungen/WareneingangView.svelte';
	import OrderRecommendations from './components/bestellungen/OrderRecommendations.svelte';
	import SupplierManager from './components/bestellungen/SupplierManager.svelte';
	import BestellHistorie from './components/bestellungen/BestellHistorie.svelte';
	import BestellBerichte from './components/bestellungen/BestellBerichte.svelte';
	import BestelllinkHinweis from './components/bestellungen/BestelllinkHinweis.svelte';
	import PrintSuggestion from './components/bestellungen/PrintSuggestion.svelte';
	import KlassensatzReservierungen from './components/bestellungen/KlassensatzReservierungen.svelte';

	let activeTab = $state('bestellungen');
	let showWareneingang = $state(false);

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
	let showGreenFade = $state(false);
	/** @type {any[] | null} Exemplare der letzten Einbuchung ohne gedrucktes Etikett */
	let printSuggestion = $state(null);

	/** Exemplare ohne Barcode-Etikett — speist den stehenden Hinweis in PrintSuggestion. */
	let offeneEtiketten = $state(0);

	async function ladeOffeneEtiketten() {
		try {
			const daten = await apiGet('/api/exemplare/etiketten-offen/anzahl');
			offeneEtiketten = daten?.anzahl ?? 0;
		} catch (err) {
			// Ein fehlender Hinweis darf das Bestellwesen nicht aufhalten.
			console.error('Offene Etiketten konnten nicht gezählt werden', err);
		}
	}

	onMount(() => {
		orderStore.init();
		// Zusätzlich zu init(): Das Laden dort ist 60 Sekunden lang gecacht. Wer die
		// öffentliche Adresse gerade in den Einstellungen nachgetragen hat und zurückkommt,
		// stünde sonst vor einem Hinweis, der bereits erledigt ist — und würde ihn beim
		// nächsten Mal nicht mehr ernst nehmen.
		orderStore.loadKonfiguration();
		ladeOffeneEtiketten();
	});

	/** @param {any[]} receivedItems */
	async function handleShipmentReceived(receivedItems) {
		showWareneingang = false;
		const needsPrinting = receivedItems.filter((item) => !item.etikett_gedruckt);
		printSuggestion = needsPrinting.length > 0 ? needsPrinting : null;
		showGreenFade = true;
		await orderStore.loadIncomingShipments();
		orderStore.loadRecommendations();
		ladeOffeneEtiketten();
		setTimeout(() => {
			showGreenFade = false;
		}, 1500);
	}

	function handlePrintSuggestion() {
		printQueue.copies = printSuggestion;
		printSuggestion = null;
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

<div class="w-full h-full text-slate-800 font-sans flex flex-col gap-6">
	<!-- Über den Reitern und nicht in einem von ihnen: Die fehlende Adresse betrifft das
	     Bestellen (Mail ohne Link) UND die Historie (Bestätigung, die nie kommt). -->
	<BestelllinkHinweis />

	<!-- Tab-Bar: reine Navigation, keine Aktionen -->
	<div class="flex items-end gap-6 border-b border-slate-200 shrink-0">
		{#snippet tab(id, label)}
			<button
				onclick={() => (activeTab = id)}
				class="pb-3 text-sm font-semibold border-b-2 transition-colors cursor-pointer {activeTab ===
				id
					? 'border-blue-600 text-blue-700'
					: 'border-transparent text-slate-500 hover:text-slate-800'}">{label}</button
			>
		{/snippet}
		{@render tab('bestellungen', 'Bestellungen')}
		{@render tab('lieferanten', 'Lieferanten verwalten')}
		{@render tab('historie', 'Bestellhistorie')}
		{@render tab('berichte', 'Berichte')}
		<button
			onclick={() => (activeTab = 'klassensaetze')}
			class="pb-3 text-sm font-semibold border-b-2 transition-colors cursor-pointer flex items-center gap-2 {activeTab ===
			'klassensaetze'
				? 'border-blue-600 text-blue-700'
				: 'border-transparent text-slate-500 hover:text-slate-800'}"
		>
			Klassensatz-Reservierungen
			{#if uiStore.pendingReservierungen > 0}
				<span
					class="min-w-5 h-5 flex items-center justify-center rounded-full bg-rose-500 text-white text-label-small font-bold px-1"
					>{uiStore.pendingReservierungen}</span
				>
			{/if}
		</button>
	</div>

	{#if activeTab === 'bestellungen'}
		{#if showWareneingang}
			<WareneingangView
				incomingShipments={orderStore.incomingShipments}
				onBack={() => (showWareneingang = false)}
				onReceived={handleShipmentReceived}
			/>
		{:else}
			<div class="flex flex-col gap-5">
				<!-- OFFENE AUFGABEN: zwei gleichwertige Statusstreifen nebeneinander.
				     Beide sagen dasselbe — „hier wartet etwas, das nichts mit der Bestellung zu
				     tun hat, an der du gerade schreibst". Der Etiketten-Hinweis stand vorher als
				     hohe Karte IN der Bestellspalte und damit über dem Warenkorb, zu dem er nicht
				     gehört: Der Wareneingang lag oben, die Etiketten rechts — dieselbe Art
				     Aufgabe an zwei Orten. -->
				<div class="grid gap-4 sm:grid-cols-2">
					<IncomingShipments
						incomingShipments={orderStore.incomingShipments}
						{showGreenFade}
						onOpenWareneingang={() => (showWareneingang = true)}
					/>
					<PrintSuggestion {printSuggestion} onPrint={handlePrintSuggestion} {offeneEtiketten} />
				</div>

				<div class="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
					<!-- HERO: der Bestellbedarf ist die tägliche Arbeitsfläche.
					     Die Spaltenbreiten stehen je Zustand VOLLSTÄNDIG da (lg und xl), statt sich
					     auf die Reihenfolge im class-Attribut zu verlassen: Bei Utilities gleicher
					     Spezifität entscheidet die Reihenfolge im Stylesheet, nicht die im Markup. -->
					<div
						class="min-w-0 {railOffen
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
							? 'lg:col-span-5 xl:col-span-4'
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
								class="hidden lg:flex w-full flex-col items-center gap-3 py-4 px-2 rounded-2xl border border-slate-200 bg-white shadow-sm hover:border-blue-400 hover:bg-blue-50/40 transition-colors cursor-pointer"
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
		{/if}
	{/if}

	{#if activeTab === 'lieferanten'}
		<SupplierManager
			suppliers={orderStore.suppliers}
			onAddSupplier={(name, email, customerNumber, istHauptlieferant) =>
				orderStore.addSupplier(name, email, customerNumber, istHauptlieferant)}
			onEditSupplier={(id, name, email, customerNumber, istHauptlieferant) =>
				orderStore.editSupplier(id, name, email, customerNumber, istHauptlieferant)}
			onRemoveSupplier={(id) => orderStore.removeSupplier(id)}
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
</div>
