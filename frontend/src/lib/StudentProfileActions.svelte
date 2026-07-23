<script>
	import Button from './components/ui/Button.svelte';
	import { apiFetch } from './apiFetch.js';
	import { idStore } from './designer/idDesignerStore.svelte.js';
	import { scale } from 'svelte/transition';
	import {
		Printer,
		FileText,
		Lock,
		Unlock,
		AlertTriangle,
		IdCard,
		ShieldCheck,
		ChevronDown,
		Layers
	} from '@lucide/svelte';

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

	// Der Ausweis-Druck ist die Primäraktion. Gibt es eine gestaltete Rückseite, bietet
	// ein Material-3-Split-Button (Hauptaktion + Chevron-Menü) die Seitenwahl — ohne die
	// Toolbar mit einem Dauer-Umschalter zuzustellen.
	const hasBack = $derived(idStore.back.elements.some((/** @type {any} */ e) => e.show));

	/** @type {{ side: 'front'|'back'|'both', label: string, hint: string, icon: any }[]} */
	const printOptions = [
		{ side: 'both', label: 'Beides', hint: 'Vorder- & Rückseite', icon: Layers },
		{ side: 'front', label: 'Nur Vorderseite', hint: 'Foto & Ausweisdaten', icon: IdCard },
		{ side: 'back', label: 'Nur Rückseite', hint: 'Hinweise & Zusatzinfos', icon: FileText }
	];

	let menuOpen = $state(false);
	/** @type {HTMLElement | null} */
	let menuAnchor = $state(null);

	/** @param {'front'|'back'|'both'} side */
	function doPrint(side) {
		menuOpen = false;
		onPrint(side);
	}

	// Menü schließt bei Klick außerhalb und mit Escape.
	$effect(() => {
		if (!menuOpen) return;
		/** @param {PointerEvent} e */
		const onDown = (e) => {
			if (menuAnchor && !menuAnchor.contains(/** @type {Node} */ (e.target))) menuOpen = false;
		};
		/** @param {KeyboardEvent} e */
		const onKey = (e) => {
			if (e.key === 'Escape') menuOpen = false;
		};
		document.addEventListener('pointerdown', onDown);
		document.addEventListener('keydown', onKey);
		return () => {
			document.removeEventListener('pointerdown', onDown);
			document.removeEventListener('keydown', onKey);
		};
	});

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

<!-- Aktionen / Dokumente — alle Druck-, Export- & Verwaltungsaktionen an einem Ort. -->
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

	<div class="flex flex-wrap gap-3 items-center">
		<!-- Primäraktion: Ausweis drucken. Mit Rückseite → Split-Button mit Seitenwahl. -->
		<div class="relative" bind:this={menuAnchor}>
			{#if hasBack}
				<div class="inline-flex rounded-full shadow-sm">
					<button
						type="button"
						onclick={() => doPrint('both')}
						class="inline-flex items-center gap-2 pl-4 pr-3.5 py-2 text-sm font-bold text-white bg-blue-600 hover:bg-blue-700 rounded-l-full transition-colors cursor-pointer"
					>
						<IdCard class="w-4 h-4" />
						Ausweis drucken
					</button>
					<button
						type="button"
						onclick={() => (menuOpen = !menuOpen)}
						aria-haspopup="menu"
						aria-expanded={menuOpen}
						aria-label="Ausweisseiten wählen"
						class="inline-flex items-center px-2.5 py-2 text-white bg-blue-600 hover:bg-blue-700 rounded-r-full border-l border-white/25 transition-colors cursor-pointer"
					>
						<ChevronDown class="w-4 h-4 transition-transform {menuOpen ? 'rotate-180' : ''}" />
					</button>
				</div>

				{#if menuOpen}
					<div
						role="menu"
						tabindex="-1"
						transition:scale={{ duration: 130, start: 0.95, opacity: 0 }}
						class="absolute left-0 top-full mt-2 z-30 w-60 origin-top-left rounded-2xl bg-white border border-slate-200 shadow-xl p-1.5"
					>
						{#each printOptions as opt (opt.side)}
							{@const Icon = opt.icon}
							<button
								type="button"
								role="menuitem"
								onclick={() => doPrint(opt.side)}
								class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-left hover:bg-slate-100 active:bg-slate-200/70 transition-colors cursor-pointer group"
							>
								<span
									class="flex items-center justify-center w-9 h-9 shrink-0 rounded-full bg-blue-50 text-blue-600 group-hover:bg-blue-100 transition-colors"
								>
									<Icon class="w-4 h-4" />
								</span>
								<span class="flex flex-col leading-tight">
									<span class="text-sm font-semibold text-slate-800">{opt.label}</span>
									<span class="text-xs text-slate-400">{opt.hint}</span>
								</span>
							</button>
						{/each}
					</div>
				{/if}
			{:else}
				<Button variant="primary" onclick={() => doPrint('both')}>
					<IdCard class="w-4 h-4" />
					Ausweis drucken
				</Button>
			{/if}
		</div>

		<!-- Kontoauszug: das (einzige) Ausleih-Dokument als archivierbares Server-PDF. -->
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

		<!-- Ersatzforderung: Rechnung an die Eltern über offene Schadensfälle. -->
		<Button
			variant="secondary"
			onclick={downloadRechnungPDF}
			disabled={rechnungPdfLoading || !profile.has_open_damages}
			title={!profile.has_open_damages
				? 'Keine offenen Schadensfälle'
				: 'Ersatzforderung über offene Schäden drucken'}
		>
			{#if rechnungPdfLoading}{@render spinner()}{:else}<AlertTriangle
					class="w-4 h-4 text-rose-600"
				/>{/if}
			Ersatzforderung
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
