<!-- @component LmfPlan — der Planer: Der Plan ist eine REIHENFOLGE von Klassen, die der
     Server auf Schultage × Stunden gießt (Peter, 05.09.2026, am echten Plan der Schule:
     Abschlussklassen zuerst, dann jeder Schultag Stunde 1–6, die Reihenfolge läuft über
     die Tage weiter). Oben der Rahmen, darunter die Reihenfolge als die bekannte Tabelle
     mit Vorschau der Plätze, unten „Nicht im Plan". Ein neuer Plan beginnt mit der
     Reihenfolge des Vorjahres. Gespeichert wird ausdrücklich — Rückgabe-Termine setzen
     Fristen. Sitzt als Reiter auf der Seite „Schuljahreswechsel". -->
<script>
	import { onMount, untrack } from 'svelte';
	import { Printer, Trash2 } from '@lucide/svelte';
	import Button from './components/ui/Button.svelte';
	import Segmente from './components/ui/Segmente.svelte';
	import LmfPlanRahmen from './components/lmfplan/LmfPlanRahmen.svelte';
	import LmfPlanReihenfolge from './components/lmfplan/LmfPlanReihenfolge.svelte';
	import LmfPlanVorrat from './components/lmfplan/LmfPlanVorrat.svelte';
	import { showToast } from '../inventur/lib/store.svelte.js';
	import * as dienst from './lmfplanDienst.js';

	let art = $state('rueckgabe');
	/** @type {import('./lmfplanDienst.js').PlanStand | null} */
	let stand = $state(null);
	/** @type {import('./lmfplanDienst.js').PlanEntwurf} */
	let entwurf = $state({
		erster_tag: '',
		startstunde: 1,
		stunden_je_tag: 6,
		zeilen: [],
		ausgelassen: []
	});
	/** @type {{ datum: string, stunde: number }[]} */
	let plaetze = $state([]);
	let laedt = $state(true);
	let speichert = $state(false);

	async function lade() {
		laedt = true;
		try {
			stand = await dienst.ladeStand(art);
			entwurf = dienst.entwurfAus(stand);
		} catch (e) {
			showToast(`${e}`, 'error');
		} finally {
			laedt = false;
		}
	}

	// Vorschau: der Server rechnet die Plätze, sobald Rahmen oder Reihenfolge sich ändern
	// (entprellt). Liest den Entwurf, schreibt NUR plaetze — kein Effekt auf eigenen State.
	$effect(() => {
		const schnappschuss = JSON.stringify({
			erster_tag: entwurf.erster_tag,
			startstunde: entwurf.startstunde,
			stunden_je_tag: entwurf.stunden_je_tag,
			n: entwurf.zeilen.length
		});
		const aktuelleArt = art;
		const timer = setTimeout(
			async () => {
				try {
					const z = await dienst.rechneVorschau(
						aktuelleArt,
						untrack(() => JSON.parse(JSON.stringify(entwurf)))
					);
					plaetze = z.map((p) => ({ datum: p.datum, stunde: p.stunde }));
				} catch (e) {
					showToast(`${e}`, 'error');
				}
			},
			schnappschuss ? 250 : 0
		);
		return () => clearTimeout(timer);
	});

	/** @param {string} k */
	function klasseRaus(k) {
		if (!entwurf.ausgelassen.some((x) => dienst.normKey(x) === dienst.normKey(k)))
			entwurf.ausgelassen = [...entwurf.ausgelassen, k].sort((a, b) =>
				a.localeCompare(b, 'de', { numeric: true })
			);
	}

	/** @param {string} k */
	function klasseHinein(k) {
		entwurf.ausgelassen = entwurf.ausgelassen.filter(
			(x) => dienst.normKey(x) !== dienst.normKey(k)
		);
		if (!entwurf.zeilen.some((z) => z.klassen.some((x) => dienst.normKey(x) === dienst.normKey(k))))
			entwurf.zeilen = [...entwurf.zeilen, { klassen: [k], vermerk: '' }];
	}

	const gueltig = $derived(
		Boolean(entwurf.erster_tag) &&
			entwurf.zeilen.every((z) => z.klassen.length > 0 || z.vermerk.trim() !== '')
	);

	async function speichern() {
		speichert = true;
		try {
			const erg = await dienst.speicherePlan(art, entwurf);
			showToast(erg.meldung, erg.ok ? 'success' : 'error');
			if (erg.ok) await lade();
		} finally {
			speichert = false;
		}
	}

	async function verwerfen() {
		if (!stand?.plan) return;
		if (!confirm(`Plan vom ${dienst.datumKurz(stand.plan.erster_tag)} verwerfen?`)) return;
		const erg = await dienst.verwerfePlan(art);
		showToast(erg.meldung, erg.ok ? 'success' : 'error');
		if (erg.ok) await lade();
	}

	async function pdf() {
		try {
			await dienst.ladePdf();
		} catch (e) {
			showToast(`${e}`, 'error');
		}
	}

	onMount(lade);
</script>

<div class="flex flex-wrap items-center justify-between gap-3">
	<Segmente
		etikett="Art des Plans"
		optionen={dienst.ARTEN.map((a) => ({ wert: a.wert, text: a.label }))}
		wert={art}
		onwahl={(/** @type {string} */ w) => {
			art = w;
			lade();
		}}
	/>
	<div class="flex items-center gap-2">
		<Button variant="secondary" onclick={pdf}>
			<Printer class="h-4 w-4" aria-hidden="true" />
			Als PDF
		</Button>
		<Button variant="secondary" onclick={verwerfen} disabled={!stand?.plan || stand.vorbei}>
			<Trash2 class="h-4 w-4" aria-hidden="true" />
			Plan verwerfen
		</Button>
		<Button onclick={speichern} disabled={!gueltig || speichert}>Plan speichern</Button>
	</div>
</div>

{#if laedt}
	<div class="flex items-center justify-center py-12">
		<div
			class="h-8 w-8 animate-spin rounded-full border-2 border-surface-container-high border-t-primary"
		></div>
	</div>
{:else}
	<p class="mt-3 text-sm text-on-surface-variant" data-testid="lmf-plan-hinweis">
		{#if stand?.plan && !stand.vorbei}
			Plan vom {dienst.datumKurz(stand.plan.erster_tag)} — Änderungen gelten nach „Plan speichern".
		{:else if stand?.plan && stand.vorbei}
			Der Plan vom {dienst.datumKurz(stand.plan.erster_tag)} ist vorbei. Dieser Entwurf übernimmt seine
			Reihenfolge — ersten Tag wählen, prüfen, speichern.
		{:else}
			Noch kein Plan. Die Reihenfolge folgt der Regel: Abschlussklassen zuerst, dann Jahrgang
			absteigend; die Oberstufe steht unter „Nicht im Plan".
		{/if}
		{#if art === 'rueckgabe'}
			Der Rückgabe-Termin einer Klasse wird die Frist ihrer Schulbücher.
		{/if}
	</p>

	<div class="mt-4 space-y-6">
		<LmfPlanRahmen
			bind:ersterTag={entwurf.erster_tag}
			bind:startstunde={entwurf.startstunde}
			bind:stundenJeTag={entwurf.stunden_je_tag}
		/>
		<LmfPlanReihenfolge bind:zeilen={entwurf.zeilen} {plaetze} onklasseraus={klasseRaus} />
		<LmfPlanVorrat klassen={entwurf.ausgelassen} onhinein={klasseHinein} />
	</div>
{/if}
