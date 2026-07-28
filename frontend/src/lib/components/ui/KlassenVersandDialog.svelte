<script>
	import { untrack } from 'svelte';
	import { Mail } from '@lucide/svelte';
	import Modal from '../../Modal.svelte';
	import Button from './Button.svelte';

	/**
	 * KlassenVersandDialog — Türsteher vor jedem klassenweisen Massenversand.
	 *
	 * Zwei Aufrufer teilen ihn sich: der Mahnlauf („Alle anmahnen") und die
	 * Abgänger-Kontoauszüge. Beide stellen dieselbe Frage — welche Klassen, und
	 * gehen die Anhänge an die Klassenleitungen oder ausnahmsweise an eine einzelne
	 * Adresse (Vertretung, Sekretariat, Probelauf). Nur die Wörter unterscheiden
	 * sich, deshalb sind sie Props und nicht zwei Komponenten: Die gefährliche
	 * Mechanik darunter (Reset beim Wiederöffnen, gesperrter Versand ohne Auswahl)
	 * darf es nur einmal geben.
	 *
	 * @type {{
	 *   open: boolean,
	 *   titel: string,
	 *   beschreibung: string,
	 *   aktion: string,
	 *   hinweis: string,
	 *   klassen?: Array<{ klasse: string, lehrer_email?: string, schueler?: unknown[] }>,
	 *   onclose: () => void,
	 *   onconfirm: (auswahl: { klassen: string[], overrideEmail: string }) => void
	 * }}
	 */
	let { open, titel, beschreibung, aktion, hinweis, klassen = [], onclose, onconfirm } = $props();

	let ausgewaehlt = $state(/** @type {string[]} */ ([]));
	let overrideEmail = $state('');

	// Der Dialog bleibt gemountet, der State überlebt also das Schließen: ohne Reset
	// trägt der nächste Lauf die Abwahl und die Fremdadresse des vorigen mit —
	// und verschickt dann still an jemand anderen als angezeigt.
	//
	// `klassen` wird bewusst untracked gelesen. Sonst würde ein Refetch des Stores bei
	// offenem Dialog (Polling, Rückkehr aus dem Offline-Sync) die gerade getroffene
	// Auswahl wieder auf „alle" zurückwerfen.
	$effect(() => {
		if (open) {
			ausgewaehlt = untrack(() => klassen.map((k) => k.klasse));
			overrideEmail = '';
		}
	});

	/** @param {string} klasse */
	function toggle(klasse) {
		ausgewaehlt = ausgewaehlt.includes(klasse)
			? ausgewaehlt.filter((k) => k !== klasse)
			: [...ausgewaehlt, klasse];
	}

	const alleGewaehlt = $derived(klassen.length > 0 && ausgewaehlt.length === klassen.length);

	// Eine vertippte Adresse würde den kompletten Lauf ins Leere schicken, ohne dass
	// jemand es merkt — der Versand bleibt deshalb gesperrt, bis sie plausibel ist.
	//
	// Erlaubt sind ZWEI Formen: der blosse Namensteil („mueller") und die vollständige
	// Adresse („extern@schulamt.hessen.de"). Alle Dienstadressen der Schule liegen auf
	// derselben Domäne; sie jedes Mal mitzutippen ist nur Fehlerquelle. Ergänzt wird
	// serverseitig aus der Absenderadresse des Systems — die Domäne steht bewusst
	// nirgends im Frontend, sonst wäre sie beim nächsten Domänenwechsel falsch.
	const emailGetrimmt = $derived(overrideEmail.trim());
	const emailOk = $derived(
		emailGetrimmt === '' ||
			/^[^\s@]+$/.test(emailGetrimmt) ||
			/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(emailGetrimmt)
	);

	const absendbar = $derived(ausgewaehlt.length > 0 && emailOk);

	/** @param {{ klasse: string, schueler?: unknown[] }} k */
	const anzahlSchueler = (k) => k.schueler?.length ?? 0;
</script>

<Modal {open} {onclose} size="md">
	{#snippet header()}
		<h3 class="text-lg font-bold text-slate-900">{titel}</h3>
	{/snippet}

	<div class="p-6 space-y-5">
		<div class="space-y-2">
			<div class="flex items-baseline justify-between gap-3">
				<p class="text-sm text-slate-600">{beschreibung}</p>
				<button
					type="button"
					class="text-xs font-semibold text-blue-600 hover:text-blue-700 shrink-0 cursor-pointer"
					onclick={() => (ausgewaehlt = alleGewaehlt ? [] : klassen.map((k) => k.klasse))}
				>
					{alleGewaehlt ? 'Keine' : 'Alle'}
				</button>
			</div>

			<div class="max-h-48 overflow-y-auto border border-slate-200 rounded-lg p-2">
				{#each klassen as k (k.klasse)}
					<label
						class="flex items-center gap-3 p-2 hover:bg-slate-50 rounded-md cursor-pointer text-sm"
					>
						<input
							type="checkbox"
							checked={ausgewaehlt.includes(k.klasse)}
							onchange={() => toggle(k.klasse)}
							class="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500/20 cursor-pointer"
						/>
						<span class="font-semibold text-slate-800">{k.klasse}</span>
						<span class="text-xs text-slate-500">{anzahlSchueler(k)} Schüler</span>
						<!-- Ohne hinterlegte Adresse überspringt der Server die Klasse still —
						     das gehört vor den Versand, nicht in die Ergebnismeldung danach. -->
						{#if !k.lehrer_email && emailGetrimmt === ''}
							<span class="ml-auto text-xs font-medium text-amber-700">keine E-Mail</span>
						{/if}
					</label>
				{:else}
					<p class="p-2 text-sm text-slate-500">Keine Klassen vorhanden.</p>
				{/each}
			</div>
		</div>

		<div>
			<label class="text-sm font-medium text-slate-700" for="versand-override-email">
				Alternative Empfänger-E-Mail (optional)
			</label>
			<input
				id="versand-override-email"
				type="text"
				inputmode="email"
				autocomplete="off"
				bind:value={overrideEmail}
				placeholder="z. B. mueller"
				aria-invalid={!emailOk}
				class="w-full h-9 px-3 mt-1 bg-white border rounded-md text-sm outline-none focus:ring-1 {emailOk
					? 'border-slate-300 focus:border-blue-600 focus:ring-blue-600'
					: 'border-rose-400 focus:border-rose-500 focus:ring-rose-500'}"
			/>
			{#if emailOk}
				<p class="text-xs text-slate-500 mt-1">{hinweis}</p>
			{:else}
				<p class="text-xs text-rose-600 mt-1">
					Keine gültige Adresse — Name oder vollständige E-Mail.
				</p>
			{/if}
		</div>
	</div>

	<div class="flex justify-end gap-3 p-4 border-t border-slate-100 bg-slate-50/50">
		<Button variant="secondary" onclick={onclose}>Abbrechen</Button>
		<Button
			variant="danger-solid"
			disabled={!absendbar}
			onclick={() => onconfirm({ klassen: ausgewaehlt, overrideEmail: emailGetrimmt })}
		>
			<Mail class="h-4 w-4" />
			{ausgewaehlt.length}
			{ausgewaehlt.length === 1 ? 'Klasse' : 'Klassen'}
			{aktion}
		</Button>
	</div>
</Modal>
