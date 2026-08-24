<script>
	import Button from './components/ui/Button.svelte';
	import AusweisGueltigkeit from './components/AusweisGueltigkeit.svelte';
	import { apiFetch } from './apiFetch.js';
	import { idStore } from './designer/idDesignerStore.svelte.js';
	import { scale } from 'svelte/transition';
	import {
		Printer,
		FileText,
		AlertTriangle,
		IdCard,
		ShieldCheck,
		ChevronDown,
		Layers
	} from '@lucide/svelte';

	/**
	 * @typedef {Object} Props
	 * @property {any} profile
	 * @property {boolean} [darfAuskunft]  manage_students_admin — DSGVO-Auskunft (Art. 15)
	 * @property {boolean} kontoauszugPdfLoading
	 * @property {boolean} rechnungPdfLoading
	 * @property {() => void} downloadKontoauszugPDF
	 * @property {() => void} downloadRechnungPDF
	 * @property {(side: 'front'|'back'|'both') => void} onPrint
	 * @property {number|null} gueltigBis        aktuell gewaehltes Ablaufjahr
	 * @property {(jahr: number|null) => void} onGueltigBis
	 */
	/** @type {Props} */
	let {
		profile,
		darfAuskunft = false,
		kontoauszugPdfLoading,
		rechnungPdfLoading,
		downloadKontoauszugPDF,
		downloadRechnungPDF,
		onPrint,
		gueltigBis,
		onGueltigBis
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
			if (e.key !== 'Escape') return;
			menuOpen = false;
			// Als verarbeitet melden, sonst schliesst derselbe Tastendruck zusätzlich das
			// Schülerprofil (globaler Escape-Kurzbefehl in Router.svelte).
			e.preventDefault();
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

<!-- Nur noch Dokumente: alles hier erzeugt ein PDF und lässt sich wegwerfen.
     Der Kasten hiess bis zum 08.08.2026 „Dokumente & Aktionen" und trug als einzige
     Aktion mit Folgen die Schülersperre — das „&" im Titel war das Eingeständnis,
     dass zwei Kategorien in einer Kiste lagen. Die Sperre steht jetzt bei dem
     Zustand, den sie umschaltet (StudentProfileCard, Konto-Status). -->
<div class="bg-slate-50 border border-slate-200 rounded-2xl p-4 shadow-sm flex flex-col gap-3">
	<h4 class="text-xs font-medium text-slate-500 flex items-center gap-1.5">
		<FileText class="w-3.5 h-3.5" />
		Dokumente
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
				<div class="inline-flex rounded-md shadow-sm">
					<Button type="button" onclick={() => doPrint('both')} class="rounded-r-none">
						<IdCard class="w-4 h-4" />
						Ausweis drucken
					</Button>
					<Button
						type="button"
						onclick={() => (menuOpen = !menuOpen)}
						aria-haspopup="menu"
						aria-expanded={menuOpen}
						aria-label="Ausweisseiten wählen"
						class="rounded-l-none border-l-white/25 px-2.5"
					>
						<ChevronDown class="w-4 h-4 transition-transform {menuOpen ? 'rotate-180' : ''}" />
					</Button>
				</div>

				{#if menuOpen}
					<div
						role="menu"
						tabindex="-1"
						transition:scale={{ duration: 130, start: 0.95, opacity: 0 }}
						class="absolute left-0 top-full mt-2 z-30 w-60 origin-top-left rounded-sm bg-surface-container shadow-xl p-1.5"
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

		<!-- Das Ablaufjahr steht NEBEN dem Druckknopf, nicht hinter ihm in einem Dialog:
		     Wer druckt, soll sehen, was auf die Karte kommt, bevor die Karte im Drucker
		     liegt. Ein Bestaetigungsdialog haette denselben Wert erst nach dem Klick. -->
		<AusweisGueltigkeit
			vorschlag={profile.ausweis_gueltig_bis ?? null}
			wert={gueltigBis}
			klasse={profile.klasse ?? ''}
			onWert={onGueltigBis}
		/>

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

		<!-- Ersatzforderung: Rechnung an die Eltern über offene Schadensfälle.

		     data-tip am UMSCHLAG, nicht am Knopf: Ein disabled-Element bekommt keine
		     Zeigerereignisse — weder für den nativen title noch für die Blase dieses
		     Projekts (tooltip.js hört delegiert auf mouseover). Die Begründung stand
		     also im Code und erreichte genau in dem Zustand niemanden, in dem man sie
		     braucht: wenn der Knopf grau ist und man wissen will, warum. Der Umschlag
		     fängt das Ereignis ab, das der graue Knopf durchlässt. -->
		<span
			class="inline-flex"
			data-tip={!profile.has_open_damages
				? 'Kein offener Schadensfall — eine Ersatzforderung gibt es erst, wenn ein Schaden erfasst ist'
				: 'Ersatzforderung über offene Schäden drucken'}
		>
			<Button
				variant="secondary"
				onclick={downloadRechnungPDF}
				disabled={rechnungPdfLoading || !profile.has_open_damages}
			>
				{#if rechnungPdfLoading}{@render spinner()}{:else}<AlertTriangle
						class="w-4 h-4 text-rose-600"
					/>{/if}
				Ersatzforderung
			</Button>
		</span>

		{#if darfAuskunft}
			<Button
				variant="secondary"
				onclick={downloadDsgvoAuskunft}
				title="DSGVO-Auskunft (Art. 15) als PDF exportieren"
			>
				<ShieldCheck class="w-4 h-4 text-slate-500" />
				DSGVO-Auskunft
			</Button>
		{/if}
	</div>
</div>
