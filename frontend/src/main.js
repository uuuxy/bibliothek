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
import * as Sentry from '@sentry/svelte';

// Keine Personendaten an Sentry. Bis Version 10 hieß das `sendDefaultPii: false`; Version 11
// hat die Option gestrichen und sammelt ohne Angabe Nutzerdaten, Cookies, Header, Bodies und
// Query-Parameter (resolveDataCollectionOptions, Vorgabe jeweils „an"). Wer die Zeile hier
// löscht, schaltet das still ein.
Sentry.init({
	dsn: import.meta.env.VITE_SENTRY_DSN,
	dataCollection: {
		userInfo: false,
		cookies: false,
		httpHeaders: false,
		httpBodies: [],
		urlQueryParams: false
	}
});

registerSW({ immediate: true });

const target = document.getElementById('app');
const app = target ? mount(App, { target }) : undefined;

export default app;
