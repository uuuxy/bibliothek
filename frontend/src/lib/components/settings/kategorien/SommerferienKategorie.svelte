<script>
	/**
	 * @component SommerferienKategorie
	 * Die Sommerferien, an denen der LMF-Planer hängt (Donnerstag davor: Ende des
	 * Büchertauschs; erster Schultag danach: Beginn der Bücherausgabe). Das Programm
	 * bringt die Jahre des aktuellen KMK-Beschlusses mit; was danach kommt, trägt die
	 * Schule hier selbst ein — die Warnung unter Betriebsbereitschaft zeigt hierher,
	 * nicht auf ein Programm-Update (Peter, 06.09.2026: „das muss doch dann irgendwo
	 * eingestellt werden"). Ein eigener Eintrag für ein Programmjahr gilt vor dem
	 * Programm (der Landesbeschluss kann sich ändern). Gespeichert wird eine JSON-Liste
	 * unter „sommerferien"; geprüft wird sie im Server (pkg/lmfplan/ferien_einstellung.go).
	 * Teil der Kategorie „LUSD & Versetzung" (SchuljahreswechselBereich).
	 */
	import { untrack } from 'svelte';
	import Tabelle from '../../ui/Tabelle.svelte';
	import { X } from '@lucide/svelte';
	import Button from '../../ui/Button.svelte';
	import Feld from '../../ui/Feld.svelte';
	import { speichereKategorie } from '../../../einstellungenSpeichern.js';

	/** @typedef {{ jahr: number, von: string, bis: string }} Eintrag */

	/** @type {{ daten: Record<string, any>, onSaved?: () => void | Promise<void> }} */
	let { daten, onSaved } = $props();

	const start = untrack(() => daten);
	/** @type {Eintrag[]} */
	const programm = start.sommerferien_programm ?? [];

	/** @param {unknown} text @returns {Eintrag[]} */
	function gelesen(text) {
		if (typeof text !== 'string' || !text.trim()) return [];
		try {
			const liste = JSON.parse(text);
			return Array.isArray(liste) ? liste : [];
		} catch {
			return [];
		}
	}

	/** @type {Eintrag[]} */
	let eigene = $state(gelesen(start.sommerferien));
	let von = $state('');
	let bis = $state('');

	// Programm und eigene Einträge in EINER Liste nach Jahr; das eigene Jahr gewinnt.
	const zeilen = $derived.by(() => {
		/** @type {Record<number, Eintrag & { eigen: boolean }>} */
		const nachJahr = {};
		for (const e of programm) nachJahr[e.jahr] = { ...e, eigen: false };
		for (const e of eigene) nachJahr[e.jahr] = { ...e, eigen: true };
		return Object.values(nachJahr).sort((a, b) => a.jahr - b.jahr);
	});

	const jahr = $derived(von ? Number(von.slice(0, 4)) : 0);
	const aufnehmbar = $derived(
		Boolean(von && bis && bis > von && bis.slice(0, 4) === von.slice(0, 4))
	);

	function aufnehmen() {
		if (!aufnehmbar) return;
		eigene = [...eigene.filter((e) => e.jahr !== jahr), { jahr, von, bis }].sort(
			(a, b) => a.jahr - b.jahr
		);
		von = '';
		bis = '';
	}

	/** @param {number} j */
	const entfernen = (j) => (eigene = eigene.filter((e) => e.jahr !== j));

	const speichern = () =>
		speichereKategorie({
			felder: { sommerferien: eigene.length ? JSON.stringify(eigene) : '' },
			onSaved
		});

	const datumFormat = new Intl.DateTimeFormat('de-DE', {
		day: '2-digit',
		month: '2-digit',
		year: 'numeric'
	});
	/** @param {string} iso */
	function kurz(iso) {
		const [j, m, t] = iso.split('-').map(Number);
		return datumFormat.format(new Date(j, m - 1, t));
	}
</script>

<section aria-labelledby="sommerferien-titel" class="space-y-4" data-testid="sommerferien">
	<div>
		<h3 id="sommerferien-titel" class="text-title-medium font-medium text-on-surface">
			Sommerferien
		</h3>
		<p class="mt-1 max-w-3xl text-sm text-on-surface-variant">
			Der LMF-Plan endet am Donnerstag vor den Sommerferien und beginnt am ersten Schultag danach.
			Das Programm kennt die Jahre des aktuellen KMK-Beschlusses; spätere Jahre trägst du hier ein
			(kmk.org/service/ferienregelung).
		</p>
	</div>

	<Tabelle class="max-w-2xl">
		<thead>
			<tr>
				<th class="w-px">Jahr</th>
				<th>Beginn</th>
				<th>Ende</th>
				<th>Quelle</th>
				<th class="w-px"><span class="sr-only">Entfernen</span></th>
			</tr>
		</thead>
		<tbody>
			{#each zeilen as z (z.jahr)}
				<tr class="h-12">
					<td class="font-medium tabular-nums">{z.jahr}</td>
					<td class="tabular-nums">{kurz(z.von)}</td>
					<td class="tabular-nums">{kurz(z.bis)}</td>
					<td>{z.eigen ? 'eigener Eintrag' : 'Programm'}</td>
					<td class="text-right">
						{#if z.eigen}
							<Button
								variant="ghost"
								size="sm"
								onclick={() => entfernen(z.jahr)}
								title="Sommerferien {z.jahr} entfernen"
								aria-label="Sommerferien {z.jahr} entfernen"
							>
								<X class="h-4 w-4" aria-hidden="true" />
							</Button>
						{/if}
					</td>
				</tr>
			{/each}
		</tbody>
	</Tabelle>

	<div class="grid max-w-2xl grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-3">
		<Feld id="sommerferien-von" label="Beginn" type="date" bind:value={von} />
		<Feld
			id="sommerferien-bis"
			label="Ende"
			type="date"
			bind:value={bis}
			ungueltig={Boolean(von && bis) && !aufnehmbar}
			hint={von && bis && !aufnehmbar ? 'Ende nach dem Beginn, im selben Jahr' : ''}
		/>
		<div class="row-span-3 grid grid-rows-subgrid gap-y-1.5">
			<span aria-hidden="true"></span>
			<Button
				variant="secondary"
				onclick={aufnehmen}
				disabled={!aufnehmbar}
				class="justify-self-start"
			>
				{jahr ? `${jahr} aufnehmen` : 'Jahr aufnehmen'}
			</Button>
		</div>
	</div>

	<div>
		<Button onclick={speichern}>Sommerferien speichern</Button>
	</div>
</section>
