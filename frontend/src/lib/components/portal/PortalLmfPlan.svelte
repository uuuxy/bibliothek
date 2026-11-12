<!-- @component PortalLmfPlan — der LMF-Plan im Kollegiums-Portal: dieselbe Tabelle wie
     in der Bibliothek, für alle gleich, immer der aktuelle Stand (kein abgelegtes PDF).
     Nur der Plan, der als nächster dran ist (Server: LMFAbfragen.LadeNaechstenPlan): bis zum
     letzten Tag des Büchertauschs dieser, danach die Ausgabe. Die Zeilen von heute sind
     hervorgehoben, im PDF nicht. Kein Erklärsatz über dem Plan: Überschrift und der Satz
     darunter sagen, was hier steht; „Als PDF" steht in der Zeile der Überschrift.
     Kein Bezug auf Klassenleitungen: Auch Fachlehrer gehen mit ihren Klassen zum
     Büchertausch, und nicht jede Klasse hat eine hinterlegte Adresse. -->
<script>
	import { onMount } from 'svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { CalendarDays, Printer } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import SuchZustand from '../ui/SuchZustand.svelte';
	import LmfPlanTabelle from '../lmfplan/LmfPlanTabelle.svelte';
	import * as dienst from '../lmfplan/lmfplanDienst.js';
	import { fehlertext } from '../../utils/fehlertext.js';
	import { localISO } from '../../utils/datum.js';
	import { abonniereNeuladen } from '../../liveEvents.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';

	/** @type {any[]} */
	let termine = $state([]);
	/** @type {number[]} */
	let eingangsjahrgaenge = $state([]);
	let laedt = $state(true);
	let fehler = $state('');
	let heute = $state(localISO(new Date()));
	// Es gilt die jüngste Anfrage: Der alte Plan stünde sonst wieder da, wenn seine Antwort zuletzt kommt.
	let ladeNr = 0;
	let geladen = false;

	async function hole() {
		const nr = ++ladeNr;
		try {
			const plan = await dienst.ladePlan();
			if (nr === ladeNr) {
				termine = plan.termine;
				eingangsjahrgaenge = plan.eingangsjahrgaenge ?? [];
				fehler = '';
				geladen = true;
			}
			return true;
		} catch (e) {
			if (nr === ladeNr) {
				// Scheitert nur das Auffrischen, bleibt der Plan stehen: Er stimmt meistens noch, und
				// die Seite läuft nebenbei, ohne dass jemand einen Fehler wegklickt.
				if (geladen)
					snackbars.addToast(`Der Plan wurde nicht aktualisiert: ${fehlertext(e)}`, 'error');
				else fehler = fehlertext(e);
			}
			return false;
		} finally {
			if (nr === ladeNr) laedt = false;
		}
	}

	function erneutLaden() {
		laedt = true;
		hole();
	}

	// Mit dem Tag wechseln die hervorgehobenen Zeilen und nach dem letzten Tag eines Plans der Plan
	// selbst. Der Rechner im Lehrerzimmer läuft über Nacht; die Minute holt auch ein aufgewachter
	// Rechner nach, dessen Zeitgeber im Schlaf stand.
	function pruefeTag() {
		const tag = localISO(new Date());
		if (tag === heute) return;
		heute = tag;
		hole();
	}

	onMount(() => {
		hole();
		// „Immer auf dem aktuellen Stand" galt bisher nur im Moment des Öffnens: Im
		// Lehrerzimmer läuft diese Seite nebenbei, und eine verschobene Stunde erschien
		// erst nach F5. Nur abonnieren, nicht verbinden — die Leitung gehört der Sitzung
		// (liveEvents.js). Das Signal trägt keine Daten; geholt wird über den eigenen
		// Endpunkt, der nur veröffentlichte Pläne kennt. Auch nach einer Unterbrechung: Der Rechner
		// im Lehrerzimmer schläft zwischendurch.
		const abbestellen = abonniereNeuladen('lmf-plan', hole);
		const takt = setInterval(pruefeTag, 60_000);
		return () => {
			abbestellen();
			clearInterval(takt);
		};
	});
</script>

{#if laedt}
	<div class="py-12 flex justify-center items-center">
		<Ladekreis size="lg" />
	</div>
{:else if fehler}
	<LadeFehler titel="LMF-Plan nicht geladen" text={fehler} onerneut={erneutLaden} />
{:else if termine.length === 0}
	<SuchZustand
		symbol={CalendarDays}
		titel="Der nächste Plan folgt"
		hinweis="Sobald die Bibliothek ihn veröffentlicht, steht er hier."
	/>
{:else}
	<LmfPlanTabelle {termine} {eingangsjahrgaenge} {heute}>
		{#snippet aktion()}
			<Button
				variant="secondary"
				onclick={() => dienst.ladePdf().catch((e) => snackbars.addToast(fehlertext(e), 'error'))}
			>
				<Printer class="h-4 w-4" aria-hidden="true" />
				Als PDF
			</Button>
		{/snippet}
	</LmfPlanTabelle>
{/if}
