<script>
	// Der Bestätigungs-Zustand einer Bestellung in der Historie.
	//
	// Der Regelweg läuft über den Link, den Bibliosys mit der Bestellmail verschickt:
	// Der Lieferant druckt dort seine Etiketten und bestätigt selbst. Deshalb zeigt
	// dieser Block zuerst, ob ein Link unterwegs ist — und erst danach den manuellen
	// Nachtrag, der nur noch Rückfallebene ist (telefonische Zusage o. Ä.).
	import { apiPut } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import Button from '../ui/Button.svelte';

	let { b, onAktualisieren } = $props();

	let laeuft = $state(false);
	// Der Link ist nur EINMAL sichtbar: In der Datenbank steht bloß sein Hash, damit ein
	// Datenbank-Auszug keine benutzbaren Links enthält. Wer ihn verliert, erzeugt einen
	// neuen — der alte stirbt dabei.
	let neuerLink = $state('');
	let kopiert = $state(false);

	async function linkErzeugen() {
		laeuft = true;
		try {
			const res = await apiPut(`/api/bestellungen/${b.id}/bestaetigungs-link`, {});
			neuerLink = res?.link || '';
			kopiert = false;
			await onAktualisieren();
		} catch {
			// Fehlermeldung kommt als Toast aus apiFetch (z. B. fehlende öffentliche Adresse).
		} finally {
			laeuft = false;
		}
	}

	async function kopieren() {
		try {
			await navigator.clipboard.writeText(neuerLink);
			kopiert = true;
		} catch {
			toastStore.addToast('Kopieren nicht möglich — Link bitte von Hand markieren.', 'error');
		}
	}

	/** @param {'klein'|'gross'} groesse */
	async function nachtragen(groesse) {
		laeuft = true;
		try {
			await apiPut(`/api/bestellungen/${b.id}/bestaetigen`, { etiketten_groesse: groesse });
		} finally {
			// Immer neu laden, auch nach einem 409: Dann hat der Lieferant selbst oder ein
			// anderer Arbeitsplatz parallel bestätigt, und die Zeile zeigte sonst weiter
			// „offen" mit toten Knöpfen.
			await onAktualisieren();
			laeuft = false;
		}
	}
</script>

<div class="mb-3 space-y-2 rounded-lg border border-slate-200 bg-white px-3 py-2.5">
	{#if b.bestaetigt_am}
		<span class="text-sm font-semibold text-emerald-700">
			✓ {b.bestaetigt_durch === 'lieferant'
				? 'Vom Lieferanten über den Link bestätigt'
				: 'Bestätigung von der Bibliothek nachgetragen'}
			{#if b.etiketten_groesse}
				({b.etiketten_groesse === 'gross' ? 'große' : 'kleine'} Etiketten)
			{/if}
		</span>
	{:else}
		<div class="flex flex-wrap items-center justify-between gap-3">
			{#if b.link_aktiv}
				<span class="text-sm font-medium text-slate-500">
					Bestätigungs-Link ist beim Lieferanten — er wählt seine Etiketten und bestätigt selbst.
				</span>
			{:else}
				<span class="text-sm font-medium text-slate-500">
					Für diese Bestellung ist kein gültiger Link unterwegs.
				</span>
			{/if}
			<Button variant="secondary" size="sm" disabled={laeuft} onclick={linkErzeugen}>
				{b.link_aktiv ? 'Neuen Link erzeugen' : 'Link erzeugen'}
			</Button>
		</div>

		{#if neuerLink}
			<div class="rounded-lg bg-slate-50 px-3 py-2.5">
				<p class="text-xs font-medium text-slate-500">
					Nur jetzt sichtbar — gespeichert wird der Link nicht, ein früherer ist ab sofort ungültig.
				</p>
				<div class="mt-1.5 flex items-center gap-2">
					<input
						readonly
						value={neuerLink}
						class="w-full rounded-lg border border-slate-200 bg-white px-2 py-1.5 font-mono text-xs text-slate-700"
					/>
					<Button variant="secondary" size="sm" onclick={kopieren}>
						{kopiert ? 'Kopiert' : 'Kopieren'}
					</Button>
				</div>
			</div>
		{/if}

		<div class="flex flex-wrap items-center justify-between gap-3 border-t border-slate-100 pt-2">
			<span class="text-xs text-slate-400">
				Rückfallebene, falls der Lieferant anders zusagt — Größe hier nachtragen:
			</span>
			<div class="flex items-center gap-2">
				<Button variant="ghost" size="sm" disabled={laeuft} onclick={() => nachtragen('klein')}>
					Kleine Etiketten
				</Button>
				<Button variant="ghost" size="sm" disabled={laeuft} onclick={() => nachtragen('gross')}>
					Große Etiketten
				</Button>
			</div>
		</div>
	{/if}
</div>
