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
// Die Schrift wird in app.css eingebunden, nur Latein und nur die zwei Stärken, die Material 3
// benutzt (Roboto 400 und 500): Die CSS-Datei eines ganzen Schnitts zieht alle Zeichensätze
// mit, von Kyrillisch bis Vietnamesisch.
import './app.css';
import App from './App.svelte';
// @ts-expect-error  virtual:pwa-register wird erst von vite-plugin-pwa zur Bauzeit erzeugt
import { registerSW } from 'virtual:pwa-register';
import { programmstand } from './lib/stores/programmstand.svelte.js';
import { darstellung } from './lib/stores/darstellung.svelte.js';

// Der Stand dieses Fensters ist der Name dieser Programmdatei; der Entwicklungsserver hat keinen.
programmstand.starte(import.meta.env.DEV ? '' : new URL(import.meta.url).pathname);
darstellung.starte();
registerSW({ immediate: true, onNeedReload: () => programmstand.neuerServiceWorker() });

const target = document.getElementById('app');
const app = target ? mount(App, { target }) : undefined;

export default app;
