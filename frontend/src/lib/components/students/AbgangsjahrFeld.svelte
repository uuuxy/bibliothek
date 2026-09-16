<!--
  @component
  AbgangsjahrFeld — das Abgangsjahr eines Schülers, an Ort und Stelle änderbar.

  Eigene Datei aus zwei Gründen: StudentProfileCard stand an der 200-Zeilen-Grenze, als
  die Akte am 16.09.2026 auch Kollegium zeigen musste — und das Abgangsjahr ist genau das
  Stück, das einem Kollegen NICHT gehört. Es steuert die DSGVO-Löschung nach dem Abgang
  von der Schule; wer nie Schüler war, hat keinen.
-->
<script>
	import { RotateCcw } from '@lucide/svelte';
	import { apiClient } from '../../apiFetch.js';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';

	/** @type {{ profile: any, darfBearbeiten?: boolean }} */
	let { profile = $bindable(), darfBearbeiten = false } = $props();

	let bearbeiten = $state(false);
	let eingabe = $state(0);
	let speichert = $state(false);
	let fehler = $state('');

	function starte() {
		eingabe = profile.abgaenger_jahr;
		fehler = '';
		bearbeiten = true;
	}

	/** Rechnet das Abgangsjahr aus der Klasse (Zwilling der Backend-Regel). @param {string} klasse */
	function ausKlasse(klasse) {
		const kl = (klasse || '').toLowerCase().trim();
		const m = kl.match(/^(\d+)(.*)/);
		if (!m) return new Date().getFullYear() + 5;
		const jahrgang = parseInt(m[1], 10);
		const suffix = m[2] || '';
		const abschluss = suffix.startsWith('h') ? 9 : jahrgang >= 11 ? 13 : 10;
		const restJahre = Math.max(0, abschluss - jahrgang);
		const jetzt = new Date();
		const basis = jetzt.getMonth() >= 7 ? jetzt.getFullYear() + 1 : jetzt.getFullYear();
		return basis + restJahre;
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
			const res = await apiClient.patch(`/api/schueler/${profile.id}`, { abgaenger_jahr: jahr });
			if (res.ok) {
				profile.abgaenger_jahr = jahr;
				bearbeiten = false;
			} else {
				const d = await res.json().catch(() => ({}));
				fehler = d.error || 'Fehler beim Speichern';
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
			onclick={() => (eingabe = ausKlasse(profile.klasse))}
			title="Automatisch aus Klasse berechnen"
			><RotateCcw class="h-3.5 w-3.5" aria-hidden="true" /> Neu berechnen</Button
		>
		<Button size="sm" onclick={speichere} disabled={speichert}>
			{speichert ? '…' : 'Speichern'}
		</Button>
		<Button variant="ghost" size="sm" onclick={() => (bearbeiten = false)}>✕</Button>
	</div>
	{#if fehler}<p class="text-xs text-rose-500 mt-1">{fehler}</p>{/if}
{:else if darfBearbeiten}
	<button
		onclick={starte}
		class="text-base text-slate-500 font-semibold hover:text-blue-600 hover:underline cursor-pointer transition-colors"
		title="Abgangsjahr bearbeiten"
	>
		Abgang {profile.abgaenger_jahr} ✎
	</button>
{:else}
	<p class="text-base text-slate-500 font-semibold">Abgang {profile.abgaenger_jahr}</p>
{/if}
