// Setzt das Farbschema dieses Rechners vor dem ersten Bild, sonst blitzte ein dunkel eingestellter
// Rechner bei jedem Laden weiß auf: Das Programm läuft erst nach dem Lesen der Seite, und
// eingebettete Skripte sperrt die Sicherheitsrichtlinie. Dieselbe Regel steht in
// src/lib/stores/darstellung.svelte.js; darstellung.test.js hält beide gleich.
(() => {
	let gespeichert = null;
	try {
		gespeichert = localStorage.getItem('bibliothek.darstellung');
	} catch {
		// Ohne Speicher gilt die Vorgabe: wie das System.
	}
	const dunkel =
		gespeichert === 'dunkel' ||
		(gespeichert !== 'hell' && matchMedia('(prefers-color-scheme: dark)').matches);
	document.documentElement.classList.toggle('schema-dunkel', dunkel);
})();
