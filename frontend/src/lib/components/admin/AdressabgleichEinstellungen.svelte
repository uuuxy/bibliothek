<script>
	/**
	 * @component AdressabgleichEinstellungen
	 * Der Zugang zum Webmail der Schule, mit dem der Abgleich das globale Adressbuch holt und Konten
	 * mit Platzhalter-Adresse ihre Adresse der Schule einträgt. Er läuft am Ersten jedes Monats von
	 * selbst; „Probelauf" zeigt nur, was er übernähme, „Jetzt abgleichen" trägt es ein.
	 */
	import { onMount } from 'svelte';
	import { apiGet, apiPost, apiPut } from '../../apiFetch.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import { meldeFormular } from '../../stores/verlassenSchutz.svelte.js';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import { ergebnisZahlen, letzterLaufText } from './adressabgleichText.js';

	// Nur das erste Laden zeigt den Hinweis; nach dem Speichern lädt die Maske still nach.
	let laedt = $state(true);
	let ladeFehler = $state(false);
	let speichert = $state(false);
	let laeuft = $state(false);
	let adresse = $state('');
	let benutzer = $state('');
	let passwort = $state('');
	let hatPasswort = $state(false);
	let gespeicherteAdresse = $state('');
	let gespeichert = $state('');
	/** @type {import('./adressabgleichText.js').AbgleichStand | null} */
	let letzterLauf = $state(null);
	/** @type {{ probelauf: boolean, stand: any, uebernahmen: any[], uebergangen: any[] } | null} */
	let ergebnis = $state(null);
	let laufFehler = $state('');

	const stand = () => [adresse, benutzer].join('\n');
	const ungespeichert = $derived(passwort !== '' || stand() !== gespeichert);
	meldeFormular(() => ungespeichert);

	async function laden() {
		try {
			const daten = await apiGet('/api/admin/settings/adressabgleich');
			adresse = daten.adresse || '';
			benutzer = daten.benutzer || '';
			hatPasswort = daten.has_password || false;
			letzterLauf = daten.letzter_lauf || null;
			gespeicherteAdresse = adresse;
			gespeichert = stand();
			ladeFehler = false;
		} catch (e) {
			console.error(e);
			ladeFehler = true;
		} finally {
			laedt = false;
		}
	}

	onMount(laden);

	async function speichere() {
		speichert = true;
		try {
			await apiPut('/api/admin/settings/adressabgleich', { adresse, benutzer, passwort });
			snackbars.addToast('Zugang zum Webmail gespeichert', 'success');
			passwort = '';
			await laden();
		} catch {
			/* apiPut meldet jeden Fehler selbst, mit dem Grund */
		} finally {
			speichert = false;
		}
	}

	/** @param {boolean} probelauf */
	async function gleicheAb(probelauf) {
		laeuft = true;
		laufFehler = '';
		ergebnis = null;
		try {
			// Länger als die zwei Minuten, nach denen der Server abbricht (internal/adressabgleich/webmail.go).
			ergebnis = await apiPost(
				'/api/admin/settings/adressabgleich/lauf',
				{ probelauf },
				{ timeoutMs: 150000 }
			);
		} catch (e) {
			laufFehler = /** @type {Error} */ (e)?.message || 'Der Abgleich ließ sich nicht ausführen';
		} finally {
			laeuft = false;
		}
		if (!probelauf) {
			const daten = await apiGet('/api/admin/settings/adressabgleich').catch(() => null);
			if (daten) letzterLauf = daten.letzter_lauf || null;
		}
	}
</script>

{#if laedt}
	<p class="text-sm text-on-surface-variant">Zugang zum Webmail wird geladen …</p>
{:else if ladeFehler}
	<LadeFehler onerneut={laden} />
{:else}
	<div
		role="group"
		aria-labelledby="adressabgleich-titel"
		class="flex w-full max-w-maske flex-col gap-6"
	>
		<div class="flex flex-col gap-1">
			<h3 id="adressabgleich-titel" class="text-base font-semibold text-on-surface">
				Adressen aus dem Webmail
			</h3>
			<p class="text-sm text-on-surface-variant">
				Trägt Konten mit Platzhalter-Adresse die Adresse der Schule aus dem globalen Adressbuch ein,
				am Ersten jedes Monats von selbst. Eine vorhandene Adresse ändert der Abgleich nie.
			</p>
		</div>

		<div class="grid grid-cols-1 gap-x-8 gap-y-6 md:grid-cols-2">
			<Feld
				bind:value={adresse}
				etikett="Webmail-Adresse"
				type="url"
				placeholder="https://webmail.schule.de"
				hint="Leer lassen, um den Abgleich abzuschalten."
				class="md:col-span-2"
			/>
			<Feld
				bind:value={benutzer}
				etikett="Benutzername"
				type="text"
				placeholder="Benutzername oder E-Mail"
			/>
			<Feld
				bind:value={passwort}
				etikett="Passwort"
				type="password"
				placeholder={hatPasswort ? '•••••••• (hinterlegt)' : 'Passwort eingeben'}
				hint="Leer lassen, um das gespeicherte Passwort nicht zu ändern."
			/>
		</div>

		<div class="flex flex-wrap items-center justify-end gap-3">
			<Button
				variant="secondary"
				onclick={() => gleicheAb(true)}
				disabled={laeuft || ungespeichert || !gespeicherteAdresse}
			>
				Probelauf
			</Button>
			<Button
				variant="secondary"
				onclick={() => gleicheAb(false)}
				disabled={laeuft || ungespeichert || !gespeicherteAdresse}
			>
				{laeuft ? 'Läuft …' : 'Jetzt abgleichen'}
			</Button>
			<Button onclick={speichere} disabled={speichert || !ungespeichert}>
				{speichert ? 'Wird gespeichert …' : 'Zugang speichern'}
			</Button>
		</div>

		<p class="text-sm text-on-surface-variant">{letzterLaufText(letzterLauf)}</p>

		{#if laufFehler}
			<div
				role="alert"
				class="rounded-xl border border-error bg-error-container px-4 py-3 text-sm text-on-error-container"
			>
				{laufFehler}
			</div>
		{:else if ergebnis}
			<div class="flex flex-col gap-3" role="status">
				<p class="text-sm font-medium text-on-surface">
					{ergebnis.probelauf ? 'Probelauf' : 'Abgeglichen'}: {ergebnisZahlen(
						ergebnis.stand,
						ergebnis.probelauf
					)}
				</p>
				{#if ergebnis.uebernahmen.length || ergebnis.uebergangen.length}
					<!-- Eine Liste nach M3: je Konto der Name und darunter Adresse und Weg oder der Grund. -->
					<ul class="divide-y divide-outline-variant border-y border-outline-variant">
						{#each ergebnis.uebernahmen as u (u.konto_id)}
							<li class="py-3">
								<p class="text-base text-on-surface">{u.name}</p>
								<p class="mt-0.5 text-sm wrap-break-word text-on-surface-variant">
									{u.adresse}, gefunden über: {u.weg}
								</p>
							</li>
						{/each}
						{#each ergebnis.uebergangen as u (u.konto_id)}
							<li class="py-3">
								<p class="text-base text-on-surface">{u.name}</p>
								<p class="mt-0.5 text-sm text-on-surface-variant">Keine Adresse: {u.grund}</p>
							</li>
						{/each}
					</ul>
				{/if}
			</div>
		{/if}
	</div>
{/if}
