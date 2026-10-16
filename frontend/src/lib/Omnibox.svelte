<script>
	import { onMount } from 'svelte';
	import ThekenKamera from './ThekenKamera.svelte';
	import OmniboxInput from './components/OmniboxInput.svelte';
	import OmniboxTreffer from './components/OmniboxTreffer.svelte';
	import OmniboxVormerkungHinweis from './components/OmniboxVormerkungHinweis.svelte';
	import OmniboxAktiverLeser from './components/OmniboxAktiverLeser.svelte';
	import OmniboxSperrHinweis from './components/OmniboxSperrHinweis.svelte';
	import OmniboxPrueflisteDialog from './components/OmniboxPrueflisteDialog.svelte';
	import OmniboxRandleuchten from './components/OmniboxRandleuchten.svelte';
	import OmniboxSchnellrueckgabe from './components/OmniboxSchnellrueckgabe.svelte';
	import LogoRelief from './components/ui/LogoRelief.svelte';
	import { omniboxStore } from './stores/omnibox.svelte.js';
	import { thekenTasten } from './scanOhneFokus.js';
	import { abonniere } from './liveEvents.js';
	import { appState } from '../inventur/lib/store.svelte.js';

	let { onSelectBook } = $props();

	let leserakte = $state(/** @type {any} */ (null));

	// Rückmeldung des Scanners: rot = blockiert oder fehlgeschlagen, grün = gebucht, orange =
	// Hinweis. Die Farbe trägt nur der Rand, die Fläche bleibt die des Feldes im Fokus, wie beim
	// Fehlerzustand von ui/Feld. Jeder Eintrag ist ein ganzer Farbsatz: Stünde er neben RUHE im
	// class-Attribut, gewönne die Regel, die im Stylesheet weiter hinten steht.
	const RUHE =
		'bg-surface-container-high border-transparent focus-within:bg-surface-container-lowest focus-within:border-primary focus-within:ring-1 focus-within:ring-primary';
	/** @type {Record<string, string>} */
	const RUECKMELDUNG = {
		green: 'bg-surface-container-lowest border-success ring-1 ring-success',
		orange: 'bg-surface-container-lowest border-warning ring-1 ring-warning',
		red: 'bg-surface-container-lowest border-error ring-1 ring-error'
	};
	const farbZustand = $derived(RUECKMELDUNG[omniboxStore.flashBorder] ?? RUHE);

	$effect(() => {
		if (appState.triggerStudentScan) {
			omniboxStore.queryVal = appState.triggerStudentScan;
			appState.triggerStudentScan = '';
			omniboxStore.submitAction(null, () => leserakte?.ladeNeu());
		}
	});

	onMount(() => {
		// Live-Aktualisierung des Schülerprofils: nur abonnieren, nicht verbinden.
		// Die Leitung gehört der Sitzung (liveEvents.js, aufgebaut vom Auth-Store) —
		// eine eigene aufzumachen hiesse, sie beim Verlassen der Ansicht auch wieder
		// zuzumachen, und das nähme sie allen anderen weg.
		const abmelden = abonniere('action', (e) => {
			try {
				const nachricht = JSON.parse(e.data);
				if (omniboxStore.activeStudent && nachricht.student_id === omniboxStore.activeStudent.id) {
					leserakte?.ladeNeu();
				}
			} catch (err) {
				console.error('SSE Parsing-Fehler in der Omnibox:', err);
			}
		});

		return abmelden;
	});

	// Fokussprung über den Store: EIN Zeitgeber mit Handle statt zweier gleicher Aufrufe.
	$effect(() => {
		if (!omniboxStore.isActive && !omniboxStore.isDropdownOpen && !omniboxStore.showCamera)
			omniboxStore.fokussiereScanfeld();
	});

	$effect(() => {
		/** @param {KeyboardEvent} e */
		function beiTaste(e) {
			if (e.key === 'Escape') {
				omniboxStore.escapeGedrueckt();
				omniboxStore.queryVal = '';
				omniboxStore.activeStudent = null;
				omniboxStore.lastFremdrueckgabe = null;
				omniboxStore.isDropdownOpen = false;
				if (omniboxStore.showCamera) {
					// Escape schliesst die Kamera. Das Abschalten selbst gehoert dem Bauteil
					// (ThekenKamera → KameraScanner): Es haelt den Strom, es raeumt ihn auf.
					stoppeKamera();
				}
			}
		}
		// Ein Scan landet im Scanfeld, auch wenn der Fokus auf einem Reiter, einem Knopf oder
		// nirgends steht; bei offener Rückfrage drückt er dort keinen Knopf. In der
		// Capture-Phase, damit Zeichen und Enter abgefangen sind, bevor ein Element sie bekommt.
		const thekenTaste = thekenTasten(omniboxStore);
		window.addEventListener('keydown', beiTaste);
		window.addEventListener('keydown', thekenTaste, true);
		return () => {
			window.removeEventListener('keydown', beiTaste);
			window.removeEventListener('keydown', thekenTaste, true);
		};
	});

	// Die Kamera: Hier steht nur, dass sie gezeigt wird. Starten, Lesen und Aufhören gehören
	// ThekenKamera.svelte — der Strom gehört dem Bauteil, das ihn anfordert.
	function starteKamera() {
		omniboxStore.showCamera = true;
	}

	function stoppeKamera() {
		omniboxStore.showCamera = false;
		// Ein Handscanner tippt blind: Ohne Fokus landet der naechste Scan im Nichts.
		setTimeout(() => document.getElementById('omnibox-input')?.focus(), 50);
	}
</script>

<OmniboxRandleuchten />

<!-- Die Suchleiste ist oben angedockt und bleibt beim Scrollen stehen (Material 3): Ein Feld,
     das man mit dem Scanner blind bedient, darf seinen Platz nicht wechseln. Der äußere
     Container trägt nur die Positionierung für das Relief; overflow-x-clip statt -hidden,
     weil hidden ihn zum Scrollbereich machte und die Leiste dann mit dem Inhalt wegrollte. -->
<div class="relative flex flex-1 flex-col w-full overflow-x-clip">
	{#if !omniboxStore.isActive}
		<!-- Nur im Ruhezustand: Sobald ein Konto geladen ist, füllt der Inhalt die Fläche,
		     und ein Wasserzeichen dahinter wäre Unruhe statt Dekoration. -->
		<LogoRelief />
	{/if}

	<!-- relative z-10: Das Relief ist absolut positioniert und läge sonst optisch über
	     diesem Inhalt (positionierte Elemente malen über nicht-positionierte). -->
	<div
		class="relative z-10 w-full mx-auto flex flex-1 flex-col items-center space-y-4 justify-start"
	>
		<!-- Die Fläche der Leiste hat die Farbe der Seite: In Ruhe ist hinter ihr nichts zu
		     sehen, beim Scrollen verdeckt sie den Inhalt, der unter ihr durchläuft. Der Umschalter
		     steht neben dem Feld; fehlt die Breite, rutscht er darunter. -->
		<div
			class="w-full sticky top-0 z-30 bg-surface-container-lowest pb-2 flex flex-wrap items-center justify-end gap-x-3 gap-y-2"
		>
			<!-- Material-3-Suchleiste: weiche Pille mit Flächen-Fokus, bewusst rounded-full und
			     48 px statt der 36-px-Control-Höhe: Das Scanfeld ist das Werkzeug des Kiosks. Der
			     Container trägt Fläche, Rahmen und Fokus. ring-inset zeichnet den Ring nach innen:
			     Die Pille liegt bündig an der Kante des Rollbereichs, außen würde er abgeschnitten.
			     `relative` bleibt: die Ergebnisliste hängt sich mit top-full daran. -->
			<form
				onsubmit={(e) => omniboxStore.submitAction(e, () => leserakte?.ladeNeu())}
				class="group relative flex flex-1 min-w-64 items-center h-12 px-5 rounded-full border ring-inset transition-colors no-print {omniboxStore.isShaking
					? 'animate-shake'
					: ''} {farbZustand}"
			>
				<OmniboxInput
					bind:queryVal={omniboxStore.queryVal}
					isDropdownOpen={omniboxStore.isDropdownOpen}
					selectedDropdownIndex={omniboxStore.selectedDropdownIndex}
					totalDropdownItems={omniboxStore.totalDropdownItems}
					isActive={omniboxStore.isActive}
					schnellrueckgabe={omniboxStore.schnellrueckgabe}
					showCamera={omniboxStore.showCamera}
					onInput={omniboxStore.handleInput}
					onSelect={(idx) => omniboxStore.selectDropdownItem(idx, onSelectBook)}
					onIndexChange={(idx) => (omniboxStore.selectedDropdownIndex = idx)}
					onEscape={() => (omniboxStore.isDropdownOpen = false)}
					onToggleCamera={omniboxStore.showCamera ? stoppeKamera : starteKamera}
				/>

				{#if omniboxStore.isDropdownOpen && omniboxStore.totalDropdownItems > 0}
					<OmniboxTreffer
						unifiedSearchResults={omniboxStore.unifiedSearchResults}
						selectedDropdownIndex={omniboxStore.selectedDropdownIndex}
						onSelect={(idx) => omniboxStore.selectDropdownItem(idx, onSelectBook)}
					/>
				{/if}
			</form>
			<OmniboxSchnellrueckgabe />

			{#if omniboxStore.errorMessage}
				<div class="mt-1 w-full p-3 bg-error text-on-error text-center">
					{omniboxStore.errorMessage}
				</div>
			{/if}
		</div>

		<!-- HTML5 Kamera-Scanner (Mobile) -->
		{#if omniboxStore.showCamera}
			<ThekenKamera
				stopCamera={stoppeKamera}
				bind:queryVal={omniboxStore.queryVal}
				submitAction={(e) => omniboxStore.submitAction(e, () => leserakte?.ladeNeu())}
			/>
		{/if}

		<OmniboxAktiverLeser bind:profil={leserakte} />
	</div>
</div>

<OmniboxVormerkungHinweis />

<OmniboxSperrHinweis onReload={() => leserakte?.ladeNeu()} />
<OmniboxPrueflisteDialog onReload={() => leserakte?.ladeNeu()} />

<style>
	/* Rütteln ohne Skalierung: Eine angedockte Leiste soll nicht aufpumpen. */
	@keyframes shake {
		0%,
		100% {
			transform: translate(0, 0);
		}
		15%,
		45%,
		75% {
			transform: translate(-8px, 0);
		}
		30%,
		60% {
			transform: translate(8px, 0);
		}
	}
	.animate-shake {
		animation: shake 0.4s cubic-bezier(0.36, 0.07, 0.19, 0.97) both;
	}
</style>
