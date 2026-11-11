<!--
  @component
  AbgangsjahrFeld — das Abgangsjahr eines Schülers, an Ort und Stelle änderbar.

  Eigene Datei: Das Abgangsjahr ist das Stück der Akte, das einem Kollegen nicht gehört. Es
  steuert die DSGVO-Löschung nach dem Abgang von der Schule; wer nie Schüler war, hat keinen.
-->
<script>
	import { RotateCcw } from '@lucide/svelte';
	import { apiClient, leseApiFehler } from '../../apiFetch.js';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import { abgangsjahrAm } from './abgangsjahr.js';

	/** @type {{ leser: any, darfBearbeiten?: boolean }} */
	let { leser = $bindable(), darfBearbeiten = false } = $props();

	let bearbeiten = $state(false);
	let eingabe = $state(0);
	let speichert = $state(false);
	let fehler = $state('');

	function starte() {
		eingabe = leser.abgaenger_jahr;
		fehler = '';
		bearbeiten = true;
	}

	async function speichere() {
		const jahr = parseInt(String(eingabe), 10);
		if (isNaN(jahr) || jahr < 2000 || jahr > 2100) {
			fehler = 'Bitte ein gültiges Jahr eingeben (2000–2100)';
			return;
		}
		speichert = true;
		fehler = '';
		try {
			const res = await apiClient.patch(`/api/schueler/${leser.id}`, { abgaenger_jahr: jahr });
			if (res.ok) {
				leser.abgaenger_jahr = jahr;
				bearbeiten = false;
			} else {
				fehler = await leseApiFehler(res, 'Fehler beim Speichern');
			}
		} catch {
			fehler = 'Netzwerkfehler';
		} finally {
			speichert = false;
		}
	}
</script>

{#if darfBearbeiten && bearbeiten}
	<div class="flex items-center gap-2 flex-wrap">
		<Feld
			type="number"
			min="2000"
			max="2100"
			bind:value={eingabe}
			aria-label="Abgangsjahr"
			feld="w-24 text-center font-bold"
		/>
		<Button
			variant="secondary"
			size="sm"
			onclick={() => (eingabe = abgangsjahrAm(leser.klasse, new Date()))}
			title="Automatisch aus Klasse berechnen"
			><RotateCcw class="h-3.5 w-3.5" aria-hidden="true" /> Neu berechnen</Button
		>
		<Button size="sm" onclick={speichere} disabled={speichert}>
			{speichert ? '…' : 'Speichern'}
		</Button>
		<Button
			variant="symbol"
			size="sm"
			aria-label="Abbrechen"
			data-tip="Abbrechen"
			onclick={() => (bearbeiten = false)}>✕</Button
		>
	</div>
	{#if fehler}<p role="alert" class="text-xs text-error mt-1">{fehler}</p>{/if}
{:else if darfBearbeiten}
	<button
		onclick={starte}
		class="text-base text-on-surface-variant font-semibold hover:underline cursor-pointer"
		title="Abgangsjahr bearbeiten"
	>
		Abgang {leser.abgaenger_jahr} ✎
	</button>
{:else}
	<p class="text-base text-on-surface-variant font-semibold">Abgang {leser.abgaenger_jahr}</p>
{/if}
