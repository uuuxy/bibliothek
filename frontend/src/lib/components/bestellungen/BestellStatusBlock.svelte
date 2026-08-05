<script>
	// Der Bestätigungs-Zustand einer Bestellung in der aufgeklappten Zeile.
	//
	// Aufbau nach Material 3: EIN Zustand, EINE Hauptaussage, EINE Hauptaktion. Der
	// bestätigte Fall ist ein getönter Container (success) — er meldet ein Ergebnis und
	// braucht keinen Knopf. Der offene Fall nennt zuerst, worauf gewartet wird, dann die
	// Aktion; der manuelle Nachtrag steht darunter als ruhige zweite Ebene, weil er nur
	// noch Rückfallebene ist, seit der Lieferant selbst bestätigen kann.
	import { apiPut } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import Button from '../ui/Button.svelte';
	import { CheckCircle2, Clock, Copy, Check } from '@lucide/svelte';

	let { b, onAktualisieren } = $props();

	let laeuft = $state(false);
	// Der Link ist nur EINMAL sichtbar: In der Datenbank steht bloß sein Hash, damit ein
	// Datenbank-Auszug keine benutzbaren Links enthält. Wer ihn verliert, erzeugt einen
	// neuen — der alte stirbt dabei.
	let neuerLink = $state('');
	let kopiert = $state(false);

	// Zusatzzeile als EIN String: Aneinandergereihte {#if}-Blöcke verlieren im Markup ihre
	// Leerzeichen — im Browser stand „05. August 2026· über den Link".
	let quittungsdetails = $derived(
		[
			b.bestaetigt_am ? langdatum(b.bestaetigt_am) : null,
			b.bestaetigt_durch === 'lieferant' ? 'über den zugeschickten Link' : null,
			b.etiketten_groesse
				? `${b.etiketten_groesse === 'gross' ? 'große' : 'kleine'} Etiketten`
				: null
		]
			.filter(Boolean)
			.join(' · ')
	);

	/** @param {string} iso */
	function langdatum(iso) {
		return new Date(iso).toLocaleDateString('de-DE', {
			day: '2-digit',
			month: 'long',
			year: 'numeric'
		});
	}

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

{#if b.bestaetigt_am}
	<!-- Getönter Container statt Kasten mit Rahmen: In M3 trägt ein erledigter Zustand
	     Fläche, keine Umrandung. -->
	<div class="mb-3 flex items-start gap-3 rounded-xl bg-emerald-50 px-4 py-3 text-emerald-900">
		<CheckCircle2 size={20} class="mt-0.5 shrink-0 text-emerald-700" aria-hidden="true" />
		<div>
			<p class="text-sm font-semibold">
				{b.bestaetigt_durch === 'lieferant'
					? 'Der Lieferant hat die Bestellung bestätigt'
					: 'Bestätigung in der Bibliothek nachgetragen'}
			</p>
			<p class="mt-0.5 text-xs text-emerald-800">{quittungsdetails}</p>
		</div>
	</div>
{:else}
	<div class="mb-3 rounded-xl border border-slate-200 bg-white">
		<div class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
			<div class="flex items-start gap-3">
				<Clock size={20} class="mt-0.5 shrink-0 text-amber-600" aria-hidden="true" />
				<div>
					<p class="text-sm font-semibold text-slate-800">Warten auf den Händler</p>
					<p class="mt-0.5 text-xs text-slate-500">
						{b.link_aktiv
							? 'Der Bestätigungs-Link ging mit der Bestellmail raus. Sobald der Händler dort bestätigt, erscheint es hier von selbst.'
							: 'Für diese Bestellung ist kein gültiger Link unterwegs.'}
					</p>
				</div>
			</div>
			<Button variant="secondary" size="sm" disabled={laeuft} onclick={linkErzeugen}>
				{b.link_aktiv ? 'Neuen Link erzeugen' : 'Link erzeugen'}
			</Button>
		</div>

		{#if neuerLink}
			<div class="mx-4 mb-3 rounded-lg bg-blue-50 px-3 py-2.5">
				<p class="text-xs font-medium text-blue-900">
					Nur jetzt sichtbar — gespeichert wird der Link nicht. Ein früherer ist ab sofort ungültig.
				</p>
				<div class="mt-1.5 flex items-center gap-2">
					<input
						readonly
						value={neuerLink}
						aria-label="Bestätigungs-Link"
						class="w-full rounded-lg border border-blue-200 bg-white px-2 py-1.5 font-mono text-xs text-slate-700"
					/>
					<Button variant="secondary" size="sm" onclick={kopieren}>
						{#if kopiert}<Check size={14} aria-hidden="true" />{:else}<Copy
								size={14}
								aria-hidden="true"
							/>{/if}
						{kopiert ? 'Kopiert' : 'Kopieren'}
					</Button>
				</div>
			</div>
		{/if}

		<!-- Zweite Ebene, bewusst leiser: Seit der Lieferant selbst bestätigt, ist der
		     Nachtrag nur noch für den Sonderfall da (telefonische Zusage). -->
		<div
			class="flex flex-wrap items-center justify-between gap-2 border-t border-slate-100 px-4 py-2"
		>
			<span class="text-xs text-slate-400">Hat der Händler anders zugesagt? Hier nachtragen:</span>
			<div class="flex items-center gap-1">
				<Button variant="ghost" size="sm" disabled={laeuft} onclick={() => nachtragen('klein')}>
					Kleine Etiketten
				</Button>
				<Button variant="ghost" size="sm" disabled={laeuft} onclick={() => nachtragen('gross')}>
					Große Etiketten
				</Button>
			</div>
		</div>
	</div>
{/if}
