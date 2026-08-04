/*
 * Dieses Programm ist freie Software: Sie können es unter den Bedingungen 
 * der European Union Public Licence (EUPL), Version 1.2 (oder jeder späteren 
 * Version, die von der Europäischen Kommission veröffentlicht wird), 
 * weitergeben und/oder modifizieren.
 * * Dieses Programm wird in der Hoffnung vertrieben, dass es nützlich sein wird, 
 * jedoch OHNE JEDE GARANTIE; auch ohne die implizite Garantie der 
 * MARKTGÄNGIGKEIT oder der EIGNUNG FÜR EINEN BESTIMMTEN ZWECK. 
 * Weitere Details finden Sie in der vollständigen EUPL 1.2.
 * * Eine Kopie der EUPL 1.2 sollte in diesem Repository unter der Datei LICENSE 
 * verfügbar sein. Andernfalls siehe: https://joinup.ec.europa.eu/collection/eupl/eupl-text-eupl-12
 */

import { mount } from 'svelte';
// Die Schrift wird in app.css eingebunden (Roboto, nur Latein 400 und 500).
// Hier standen bis zum 04.08.2026 SIEBEN Inter-Schnitte (300–900), und jede
// dieser CSS-Dateien zieht ALLE Zeichensätze mit — Latein, Latein erweitert,
// Kyrillisch, Griechisch, Vietnamesisch. Gemessen im Build: 51 Schriftdateien,
// 776 KB, die bei jedem Aufruf mit ausgeliefert wurden, obwohl die Anwendung
// deutschsprachig ist und Material 3 nur zwei Stärken benutzt.
import './app.css';
import App from './App.svelte';
// @ts-expect-error  virtual:pwa-register wird erst von vite-plugin-pwa zur Bauzeit erzeugt
import { registerSW } from 'virtual:pwa-register';
import * as Sentry from '@sentry/svelte';

Sentry.init({
	dsn: import.meta.env.VITE_SENTRY_DSN,
	sendDefaultPii: false
});

registerSW({ immediate: true });

const target = document.getElementById('app');
const app = target ? mount(App, { target }) : undefined;

export default app;
