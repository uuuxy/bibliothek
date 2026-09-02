<script>
	import { onMount } from 'svelte';
	import StudentProfile from './StudentProfile.svelte';
	import CameraScanner from './CameraScanner.svelte';
	import OmniboxInput from './components/OmniboxInput.svelte';
	import OmniboxResults from './components/OmniboxResults.svelte';
	import OmniboxTeacherCard from './OmniboxTeacherCard.svelte';
	import OmniboxVormerkungAlert from './components/OmniboxVormerkungAlert.svelte';
	import OmniboxThekeHinweise from './components/OmniboxThekeHinweise.svelte';
	import OmniboxBlockAlert from './components/OmniboxBlockAlert.svelte';
	import OmniboxChecklistDialog from './components/OmniboxChecklistDialog.svelte';
	import OmniboxScreenFlash from './components/OmniboxScreenFlash.svelte';
	import LogoRelief from './components/ui/LogoRelief.svelte';
	import { omniboxStore } from './stores/omnibox.svelte.js';
	import { abonniere } from './liveEvents.js';
	import { appState } from '../inventur/lib/store.svelte.js';

	let { onSelectBook } = $props();

	let studentProfileComponent = $state(/** @type {any} */ (null));

	// Rückmeldung des Scanners (rot = blockiert/fehlgeschlagen, grün = gebucht, orange = Hinweis).
	//
	// Entscheidend ist, dass Grundzustand und Rückmeldung sich AUSSCHLIESSEN, statt
	// nebeneinander im class-Attribut zu stehen: Tailwind-Utilities haben alle dieselbe
	// Spezifität, es gewinnt die Regel, die im Stylesheet WEITER HINTEN steht — nicht die,
	// die im Attribut später kommt. `border-transparent` steht dort hinter `border-red-500`,
	// also hätte der Grundzustand die Fehlerfarbe geschluckt. (Genau so gemessen: Klassen
	// gesetzt, berechnete Rahmenfarbe trotzdem transparent.)
	//
	// Deshalb liefert dieser Ausdruck den KOMPLETTEN Farbsatz — entweder Ruhe oder Rückmeldung.
	// Ein Fokus-Blau während einer roten Rückmeldung gibt es damit gar nicht erst.
	const RUHE =
		'bg-slate-100 border-transparent focus-within:bg-white focus-within:border-blue-600 focus-within:ring-1 focus-within:ring-blue-600';
	const farbZustand = $derived.by(() => {
		if (omniboxStore.flashBorder === 'green')
			return 'bg-emerald-50 border-emerald-400 ring-1 ring-emerald-400';
		if (omniboxStore.flashBorder === 'orange')
			return 'bg-amber-50 border-amber-400 ring-1 ring-amber-400';
		if (omniboxStore.flashBorder === 'red') return 'bg-red-50 border-red-500 ring-1 ring-red-500';
		return RUHE;
	});

	$effect(() => {
		if (appState.triggerStudentScan) {
			omniboxStore.queryVal = appState.triggerStudentScan;
			appState.triggerStudentScan = '';
			omniboxStore.submitAction(null, () => studentProfileComponent?.reloadProfile());
		}
	});

	onMount(() => {
		// Live-Aktualisierung des Schülerprofils: nur abonnieren, nicht verbinden.
		// Die Leitung gehört der Sitzung (liveEvents.js, aufgebaut vom Auth-Store) —
		// eine eigene aufzumachen hiesse, sie beim Verlassen der Ansicht auch wieder
		// zuzumachen, und das nähme sie allen anderen weg.
		const abmelden = abonniere('action', (e) => {
			try {
				const actionData = JSON.parse(e.data);
				if (omniboxStore.activeStudent && actionData.student_id === omniboxStore.activeStudent.id) {
					studentProfileComponent?.reloadProfile();
				}
			} catch (err) {
				console.error('SSE Parsing-Fehler in der Omnibox:', err);
			}
		});

		// Offline / Online Erkennung is now handled globally in offlineSync.svelte.js

		return abmelden;
	});

	$effect(() => {
		if (!omniboxStore.isActive && !omniboxStore.isDropdownOpen && !omniboxStore.showCamera) {
			setTimeout(() => document.getElementById('omnibox-input')?.focus(), 50);
		}
	});

	$effect(() => {
		/** @param {KeyboardEvent} e */
		function handleKeyDown(e) {
			if (e.key === 'Escape') {
				omniboxStore.queryVal = '';
				omniboxStore.activeStudent = null;
				omniboxStore.activeTeacher = null;
				omniboxStore.lastFremdrueckgabe = null;
				omniboxStore.isDropdownOpen = false;
				if (omniboxStore.showCamera) {
					omniboxStore.showCamera = false;
					if (omniboxStore.cameraScanner) {
						try {
							omniboxStore.cameraScanner.stop();
						} catch {}
						try {
							omniboxStore.cameraScanner.clear();
						} catch {}
						omniboxStore.cameraScanner = null;
					}
					setTimeout(() => document.getElementById('omnibox-input')?.focus(), 50);
				}
			}
		}
		window.addEventListener('keydown', handleKeyDown);
		return () => window.removeEventListener('keydown', handleKeyDown);
	});

	// HTML5 Kamera-Scanner (Mobile)
	async function startCamera() {
		omniboxStore.showCamera = true;
		await new Promise((r) => setTimeout(r, 80));
		try {
			const { Html5Qrcode } = await import('html5-qrcode');
			omniboxStore.cameraScanner = new Html5Qrcode('camera-scan-region');
			await omniboxStore.cameraScanner.start(
				{ facingMode: 'environment' },
				{ fps: 10, qrbox: { width: 260, height: 120 } },
				(/** @type {string} */ decodedText) => {
					omniboxStore.queryVal = decodedText.trim();
					stopCamera();
					omniboxStore.submitAction(null, () => studentProfileComponent?.reloadProfile());
				},
				() => {}
			);
		} catch {
			omniboxStore.showCamera = false;
			omniboxStore.showToast('Kamera konnte nicht gestartet werden', 'error');
		}
	}

	async function stopCamera() {
		omniboxStore.showCamera = false;
		if (omniboxStore.cameraScanner) {
			try {
				await omniboxStore.cameraScanner.stop();
			} catch {}
			try {
				omniboxStore.cameraScanner.clear();
			} catch {}
			omniboxStore.cameraScanner = null;
		}
		setTimeout(() => document.getElementById('omnibox-input')?.focus(), 50);
	}
</script>

<OmniboxScreenFlash />

<!-- ── Offline / Queue banner was replaced by global OfflineIndicator ── -->

<!-- Die Suchleiste ist oben ANGEDOCKT und bleibt es (Material 3): Sie ist ein
     persistentes Element, aus dem die Ergebnisse aufklappen — sie wechselt nicht selbst
     die Position. Vorher stand sie im Ruhezustand mittig (min-h-[60vh] justify-center)
     und sprang beim ersten Scan nach oben. Schon die frühere Animation dieses Sprungs
     war Wartezeit; der Sprung selbst ist es auch, nur kürzer. Ein Feld, das man mit dem
     Scanner blind bedient, darf seinen Platz nicht wechseln.
     Der äußere Container trägt nur die Positionierung für das Relief. -->
<div class="relative flex flex-1 flex-col w-full overflow-x-hidden">
	{#if !omniboxStore.isActive}
		<!-- Nur im Ruhezustand: Sobald ein Konto geladen ist, füllt der Inhalt die Fläche,
		     und ein Wasserzeichen dahinter wäre Unruhe statt Dekoration. -->
		<LogoRelief />
	{/if}

	<!-- relative z-10: Das Relief ist absolut positioniert und läge sonst optisch ÜBER
	     diesem Inhalt (positionierte Elemente malen über nicht-positionierte). -->
	<div
		class="relative z-10 w-full mx-auto flex flex-1 flex-col items-center space-y-6 pt-4 justify-start"
	>
		<div class="w-full sticky -top-4 z-30 bg-slate-50 py-4">
			<!-- Material-3-Suchleiste: weiche Pille mit Flächen-Fokus. Bewusst rounded-full und
		     bewusst 48 px statt der 36-px-Control-Höhe — das Scanfeld ist das globale Werkzeug
		     des Kiosks und soll sich von den eckigen Datenfeldern abheben. Der Container trägt
		     Fläche, Rahmen und Fokus; Lupe, Feld und Kamera-Knopf sind seine Flex-Kinder.
		     `relative` bleibt: die Ergebnisliste hängt sich mit top-full daran. -->
			<form
				onsubmit={(e) =>
					omniboxStore.submitAction(e, () => studentProfileComponent?.reloadProfile())}
				class="group relative flex items-center w-full h-12 px-5 rounded-full border transition-colors no-print {omniboxStore.isShaking
					? 'animate-shake'
					: ''} {farbZustand}"
			>
				<OmniboxInput
					bind:queryVal={omniboxStore.queryVal}
					isDropdownOpen={omniboxStore.isDropdownOpen}
					totalDropdownItems={omniboxStore.totalDropdownItems}
					isActive={omniboxStore.isActive}
					showCamera={omniboxStore.showCamera}
					onInput={omniboxStore.handleInput}
					onSelect={(idx) => omniboxStore.selectDropdownItem(idx, onSelectBook)}
					onIndexChange={(idx) => (omniboxStore.selectedDropdownIndex = idx)}
					onEscape={() => (omniboxStore.isDropdownOpen = false)}
					onToggleCamera={omniboxStore.showCamera ? stopCamera : startCamera}
				/>

				{#if omniboxStore.isDropdownOpen && omniboxStore.totalDropdownItems > 0}
					<OmniboxResults
						unifiedSearchResults={omniboxStore.unifiedSearchResults}
						selectedDropdownIndex={omniboxStore.selectedDropdownIndex}
						onSelect={(idx) => omniboxStore.selectDropdownItem(idx, onSelectBook)}
					/>
				{/if}
			</form>

			{#if omniboxStore.errorMessage}
				<div class="mt-3 p-3 bg-red-600 text-white text-center">
					{omniboxStore.errorMessage}
				</div>
			{/if}
		</div>

		<!-- HTML5 Kamera-Scanner (Mobile) -->
		{#if omniboxStore.showCamera}
			<CameraScanner
				{stopCamera}
				bind:queryVal={omniboxStore.queryVal}
				submitAction={(e) =>
					omniboxStore.submitAction(e, () => studentProfileComponent?.reloadProfile())}
			/>
		{/if}

		{#if omniboxStore.activeStudent}
			<!-- Fremdrückgabe- und Abholfach-Banner (200-Zeilen-Regel: eigene Datei). -->
			<OmniboxThekeHinweise />
			<StudentProfile
				bind:this={studentProfileComponent}
				student={omniboxStore.activeStudent}
				defaultTab="ausleihen"
				onDeselect={() => {
					omniboxStore.activeStudent = null;
					omniboxStore.lastFremdrueckgabe = null;
				}}
				onReturnClick={(barcode) => {
					omniboxStore.queryVal = barcode;
					omniboxStore.submitAction(null, () => studentProfileComponent?.reloadProfile());
				}}
			/>
		{:else if omniboxStore.activeTeacher}
			<OmniboxTeacherCard
				teacher={omniboxStore.activeTeacher}
				onDeselect={() => {
					omniboxStore.activeTeacher = null;
					omniboxStore.lastFremdrueckgabe = null;
				}}
			/>
		{/if}
	</div>
</div>

<!-- Toasts laufen ausschließlich über ToastContainer.svelte (global eingehängt).
     Hier stand bis zuletzt ein zweites, eigenes Toast-Markup mit anderer Optik und
     anderem Timing — zwei Meldungswege für dieselbe Sache. -->

<OmniboxVormerkungAlert />

<OmniboxBlockAlert onReload={() => studentProfileComponent?.reloadProfile()} />
<OmniboxChecklistDialog onReload={() => studentProfileComponent?.reloadProfile()} />

<style>
	/* ── Shake animation ───────────────────────────────────────
	   Eine Variante, nicht mehr zwei. Vorher gab es `shake` (scale 1.05) für die mittige
	   Ruhelage und `activeShake` (scale 1) für die angedockte — ausgewählt über den
	   Selektor `:global(.pt-4) .animate-shake`, also über das Vorhandensein einer
	   Utility-Klasse. Ein Umbenennen dieser Klasse hätte die Animation still auf die
	   falsche Variante geworfen. Seit die Leiste dauerhaft oben andockt, gibt es nur noch
	   einen Zustand: ohne Skalierung — eine angedockte Leiste soll nicht aufpumpen. */
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
