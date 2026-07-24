<!-- @component StatsTrendChart — Aktivitäts-Zeitreihe (Ausleihen + Rückgaben je Monat,
     letzte 12 Monate) als schlankes Inline-SVG. Zwei Serien derselben Einheit → EINE
     gemeinsame Y-Achse (kein Dual-Axis). Farben: erste zwei validierte Kategorial-Slots
     (Blau/Orange, CVD-geprüft). Legende immer vorhanden, Hover-Tooltip je Monat, und
     eine visuell versteckte Tabelle als barrierefreier „Table-View". Keine Personendaten. -->
<script>
	/** @typedef {{ monat: string, ausleihen: number, rueckgaben: number }} TrendPunkt */
	/** @type {{ data?: TrendPunkt[] }} */
	let { data = [] } = $props();

	// Light-Surface (die App hat keinen Dark-Mode) — Slots 1 & 2 der validierten Palette.
	const FARBE_AUSLEIHEN = '#2a78d6';
	const FARBE_RUECKGABEN = '#eb6834';

	const MONATE_KURZ = ['Jan', 'Feb', 'Mär', 'Apr', 'Mai', 'Jun', 'Jul', 'Aug', 'Sep', 'Okt', 'Nov', 'Dez'];
	/** @param {string} ym  "2026-07" → "Jul" */
	const monatLabel = (ym) => MONATE_KURZ[Number(ym.split('-')[1]) - 1] ?? ym;
	/** @param {string} ym  "2026-07" → "2026" */
	const jahr = (ym) => ym.split('-')[0];

	// Geometrie in viewBox-Einheiten; das SVG skaliert per width:100% mit dem Container.
	const VBW = 720;
	const VBH = 240;
	const M = { top: 12, right: 12, bottom: 28, left: 40 };
	const plotW = VBW - M.left - M.right;
	const plotH = VBH - M.top - M.bottom;
	const baselineY = M.top + plotH;

	const hatDaten = $derived(data.some((d) => d.ausleihen > 0 || d.rueckgaben > 0));
	const maxVal = $derived(Math.max(1, ...data.flatMap((d) => [d.ausleihen, d.rueckgaben])));

	/** „Schöne" Schrittweite (1/2/5·10ⁿ), damit Y-Ticks ganzzahlig bleiben. @param {number} max */
	function niceStep(max) {
		const raw = max / 4;
		const pow = Math.pow(10, Math.floor(Math.log10(raw)));
		const n = raw / pow;
		return (n <= 1 ? 1 : n <= 2 ? 2 : n <= 5 ? 5 : 10) * pow;
	}
	const step = $derived(niceStep(maxVal));
	const yMax = $derived(Math.ceil(maxVal / step) * step);

	const yTicks = $derived.by(() => {
		const ticks = [];
		for (let v = 0; v <= yMax + 1e-9; v += step) {
			ticks.push({ val: v, y: baselineY - (v / yMax) * plotH });
		}
		return ticks;
	});

	const groups = $derived.by(() => {
		const n = data.length || 1;
		const groupW = plotW / n;
		const barW = Math.min(13, groupW * 0.28);
		const gap = 3; // 2px+ Surface-Gap zwischen benachbarten Balken (Mark-Spec)
		const pairW = barW * 2 + gap;
		return data.map((d, i) => {
			const cx = M.left + groupW * (i + 0.5);
			const x1 = cx - pairW / 2;
			const yA = baselineY - (d.ausleihen / yMax) * plotH;
			const yR = baselineY - (d.rueckgaben / yMax) * plotH;
			return {
				i,
				...d,
				cx,
				groupX: M.left + groupW * i,
				groupW,
				barA: { x: x1, y: yA, h: baselineY - yA, w: barW },
				barR: { x: x1 + barW + gap, y: yR, h: baselineY - yR, w: barW }
			};
		});
	});

	/** Balken mit nur oben gerundeten Ecken, an der Baseline verankert (Mark-Spec). */
	function barPath(/** @type {{x:number,y:number,w:number,h:number}} */ b) {
		if (b.h <= 0.5) return ''; // Nullwert → nichts zeichnen
		const r = Math.min(2.5, b.w / 2, b.h);
		const bottom = b.y + b.h;
		return `M${b.x},${bottom} L${b.x},${b.y + r} Q${b.x},${b.y} ${b.x + r},${b.y} L${b.x + b.w - r},${b.y} Q${b.x + b.w},${b.y} ${b.x + b.w},${b.y + r} L${b.x + b.w},${bottom} Z`;
	}

	let hoverIdx = $state(-1);
	const hovered = $derived(hoverIdx >= 0 ? groups[hoverIdx] : null);

	// Ein einziger Handler am SVG (statt Treffer-Rechtecke je Monat): rechnet aus der
	// Zeigerposition den Monatsindex. Die volle Datenlage liegt zusätzlich in der
	// sr-only-Tabelle unten — der Hover ist reine Maus-Ergänzung.
	function onMove(/** @type {MouseEvent} */ e) {
		const el = /** @type {SVGSVGElement} */ (e.currentTarget);
		const rect = el.getBoundingClientRect();
		const vbX = ((e.clientX - rect.left) / rect.width) * VBW;
		const n = data.length || 1;
		const idx = Math.floor((vbX - M.left) / (plotW / n));
		hoverIdx = idx >= 0 && idx < n ? idx : -1;
	}
</script>

<div class="w-full">
	<!-- Kopf: Titel + Legende (Legende immer vorhanden → Identität nie farb-only) -->
	<div class="flex flex-wrap items-baseline justify-between gap-2 mb-1">
		<h3 class="font-bold text-slate-700 text-sm uppercase tracking-wider font-sans">
			Aktivität pro Monat
		</h3>
		<div class="flex items-center gap-4 text-xs font-semibold">
			<span class="flex items-center gap-1.5 text-slate-600">
				<span class="w-2.5 h-2.5 rounded-sm" style="background:{FARBE_AUSLEIHEN}"></span>Ausleihen
			</span>
			<span class="flex items-center gap-1.5 text-slate-600">
				<span class="w-2.5 h-2.5 rounded-sm" style="background:{FARBE_RUECKGABEN}"></span>Rückgaben
			</span>
		</div>
	</div>
	<p class="text-xs text-slate-400 mb-2">Letzte 12 Monate · nach Ausleih- bzw. Rückgabedatum</p>

	{#if !hatDaten}
		<div class="py-12 text-center text-xs text-slate-400 font-medium">
			<span class="text-2xl block mb-2">📈</span>
			Noch keine Ausleih-Aktivität im letzten Jahr.
		</div>
	{:else}
		<div class="relative">
			<svg
				viewBox="0 0 {VBW} {VBH}"
				class="w-full h-auto"
				role="img"
				aria-label="Balkendiagramm: Ausleihen und Rückgaben je Monat über die letzten 12 Monate. Details in der folgenden Tabelle."
				onmousemove={onMove}
				onmouseleave={() => (hoverIdx = -1)}
			>
				<!-- Y-Gridlines + Ticks (recessiv) -->
				{#each yTicks as t, _i (_i)}
					<line x1={M.left} y1={t.y} x2={VBW - M.right} y2={t.y} stroke="#e2e8f0" stroke-width="1" />
					<text
						x={M.left - 6}
						y={t.y}
						text-anchor="end"
						dominant-baseline="middle"
						class="fill-slate-400"
						style="font-size:11px"
						font-variant-numeric="tabular-nums">{t.val.toLocaleString('de-DE')}</text
					>
				{/each}

				{#each groups as g, _i (_i)}
					<!-- Hover-Highlight des aktiven Monats -->
					{#if hoverIdx === g.i}
						<rect x={g.groupX} y={M.top} width={g.groupW} height={plotH} fill="#0f172a" opacity="0.04" />
					{/if}
					<path d={barPath(g.barA)} fill={FARBE_AUSLEIHEN} />
					<path d={barPath(g.barR)} fill={FARBE_RUECKGABEN} />
					<!-- Monats-Label -->
					<text
						x={g.cx}
						y={baselineY + 16}
						text-anchor="middle"
						class="fill-slate-400"
						style="font-size:11px">{monatLabel(g.monat)}</text
					>
				{/each}

				<!-- Baseline -->
				<line
					x1={M.left}
					y1={baselineY}
					x2={VBW - M.right}
					y2={baselineY}
					stroke="#cbd5e1"
					stroke-width="1"
				/>
			</svg>

			{#if hovered}
				<div
					class="pointer-events-none absolute z-10 -translate-x-1/2 -top-1 rounded-lg bg-slate-900 text-white px-3 py-2 shadow-lg text-xs whitespace-nowrap"
					style="left:{(hovered.cx / VBW) * 100}%"
				>
					<div class="font-bold mb-1">{monatLabel(hovered.monat)} {jahr(hovered.monat)}</div>
					<div class="flex items-center gap-1.5">
						<span class="w-2 h-2 rounded-sm" style="background:{FARBE_AUSLEIHEN}"></span>
						Ausleihen: <span class="font-bold tabular-nums">{hovered.ausleihen}</span>
					</div>
					<div class="flex items-center gap-1.5">
						<span class="w-2 h-2 rounded-sm" style="background:{FARBE_RUECKGABEN}"></span>
						Rückgaben: <span class="font-bold tabular-nums">{hovered.rueckgaben}</span>
					</div>
				</div>
			{/if}
		</div>

		<!-- Barrierefreier Table-View (visuell versteckt): identisch zu den Balken -->
		<table class="sr-only">
			<caption>Ausleihen und Rückgaben je Monat, letzte 12 Monate</caption>
			<thead>
				<tr><th>Monat</th><th>Ausleihen</th><th>Rückgaben</th></tr>
			</thead>
			<tbody>
				{#each data as d, _i (_i)}
					<tr>
						<td>{monatLabel(d.monat)} {jahr(d.monat)}</td>
						<td>{d.ausleihen}</td>
						<td>{d.rueckgaben}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</div>
