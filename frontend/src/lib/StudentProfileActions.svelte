<script>
	import Button from './components/ui/Button.svelte';
	import { apiFetch } from './apiFetch.js';
	import { idStore } from './designer/idDesignerStore.svelte.js';
	import { Printer, FileText, Lock, Unlock, AlertTriangle, IdCard, ShieldCheck } from '@lucide/svelte';

	/**
	 * @typedef {Object} Props
	 * @property {any} profile
	 * @property {string} role
	 * @property {boolean} kontoauszugPdfLoading
	 * @property {boolean} rechnungPdfLoading
	 * @property {() => void} downloadKontoauszugPDF
	 * @property {() => void} downloadRechnungPDF
	 * @property {() => void} showLockModal
	 * @property {(side: 'front'|'back'|'both') => void} onPrint
	 */
	/** @type {Props} */
	let {
		profile,
		role = '',
		kontoauszugPdfLoading,
		rechnungPdfLoading,
		downloadKontoauszugPDF,
		downloadRechnungPDF,
		showLockModal,
		onPrint
	} = $props();

	// Seitenwahl für den Ausweis-Einzeldruck. Der Umschalter erscheint nur, wenn die
	// Rückseite überhaupt Inhalt hat — sonst gibt es nur die Vorderseite zu drucken.
	const hasBack = $derived(idStore.back.elements.some((/** @type {any} */ e) => e.show));
	/** @type {'front'|'back'|'both'} */
	let printSide = $state('both');

	async function downloadDsgvoAuskunft() {
		try {
			const res = await apiFetch(`/api/schueler/${profile.id}/dsgvo-auskunft/pdf`);
			if (res.ok) {
				const blob = await res.blob();
				const url = URL.createObjectURL(blob);
				const a = document.createElement('a');
				a.href = url;
				a.download = `dsgvo-auskunft-${profile.nachname || 'Unbekannt'}-${profile.vorname || 'Unbekannt'}.pdf`;
				document.body.appendChild(a);
				a.click();
				document.body.removeChild(a);
				URL.revokeObjectURL(url);
			} else {
				const text = await res.text();
				console.error('Auskunft Error:', text);
				alert('Fehler beim Herunterladen der Auskunft.');
			}
		} catch (e) {
			console.error('Netzwerkfehler DSGVO Auskunft:', e);
			alert('Netzwerkfehler');
		}
	}
</script>

<!-- Aktionen / Dokumente — alle Druck-, Export- & Verwaltungsaktionen an einem Ort.
     Der Ausweis-Druck lebt bewusst hier (nicht in der Identitätsspalte). -->
<div class="bg-slate-50 border border-slate-200 rounded-2xl p-4 shadow-sm flex flex-col gap-3">
	<h4 class="text-xs font-bold text-slate-500 uppercase tracking-wider flex items-center gap-1.5">
		<FileText class="w-3.5 h-3.5" />
		Dokumente & Aktionen
	</h4>
	{#snippet spinner()}
		<div
			class="w-4 h-4 border-2 border-slate-400 border-t-slate-700 rounded-full animate-spin"
		></div>
	{/snippet}

	{#if hasBack}
		<!-- Seitenwahl: nur relevant, wenn eine Rückseite gestaltet ist -->
		<div class="flex items-center gap-2">
			<span class="text-xs font-semibold text-slate-500">Ausweisseiten</span>
			<div
				class="flex gap-0.5 rounded-full bg-slate-100 p-0.5"
				role="group"
				aria-label="Zu druckende Ausweisseite"
			>
				{#each [['both', 'Beides'], ['front', 'Vorderseite'], ['back', 'Rückseite']] as [wert, label] (wert)}
					<button
						type="button"
						onclick={() => (printSide = /** @type {'front'|'back'|'both'} */ (wert))}
						aria-pressed={printSide === wert}
						class="px-3 py-1 text-xs font-bold rounded-full transition-colors cursor-pointer {printSide ===
						wert
							? 'bg-white text-blue-600 shadow-sm'
							: 'text-slate-500 hover:text-slate-700'}"
					>
						{label}
					</button>
				{/each}
			</div>
		</div>
	{/if}

	<div class="flex flex-wrap gap-3 items-center">
		<!-- Ausweis-Druck: Anker der Gruppe (häufigste Aktion beim Onboarding). -->
		<Button variant="primary" onclick={() => onPrint(hasBack ? printSide : 'both')}>
			<IdCard class="w-4 h-4" />
			Ausweis drucken
		</Button>

		<!-- Druck- & Export-Aktionen -->
		<Button
			variant="secondary"
			onclick={downloadKontoauszugPDF}
			disabled={kontoauszugPdfLoading || !(profile.entliehene_buecher?.length > 0)}
		>
			{#if kontoauszugPdfLoading}{@render spinner()}{:else}<Printer
					class="w-4 h-4 text-blue-600"
				/>{/if}
			Kontoauszug
		</Button>

		<Button
			variant="secondary"
			onclick={downloadRechnungPDF}
			disabled={rechnungPdfLoading || !profile.has_open_damages}
			title={!profile.has_open_damages ? 'Keine offenen Forderungen' : 'Ersatzforderung drucken'}
		>
			{#if rechnungPdfLoading}{@render spinner()}{:else}<AlertTriangle
					class="w-4 h-4 text-rose-600"
				/>{/if}
			Forderung
		</Button>

		<Button
			variant="secondary"
			onclick={() => window.print()}
			disabled={!(profile.entliehene_buecher?.length > 0)}
			title={!(profile.entliehene_buecher?.length > 0)
				? 'Keine offenen Ausleihen'
				: 'Druckansicht der Ausleihen'}
		>
			<Printer class="w-4 h-4 text-slate-500" />
			Ausleihen-Liste
		</Button>

		{#if role === 'admin'}
			<Button
				variant="secondary"
				onclick={downloadDsgvoAuskunft}
				title="DSGVO-Auskunft (Art. 15) als PDF exportieren"
			>
				<ShieldCheck class="w-4 h-4 text-slate-500" />
				DSGVO-Auskunft
			</Button>
		{/if}

		<!-- Sperr-Aktion: optisch getrennt ganz nach rechts -->
		<Button
			class="ml-auto"
			variant={profile.is_manually_blocked ? 'success' : 'danger'}
			onclick={showLockModal}
		>
			{#if profile.is_manually_blocked}
				<Unlock class="w-4 h-4" /> Sperre aufheben
			{:else}
				<Lock class="w-4 h-4" /> Schüler sperren
			{/if}
		</Button>
	</div>
</div>
