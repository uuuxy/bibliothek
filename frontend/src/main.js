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
import '@fontsource/inter/300.css';
import '@fontsource/inter/400.css';
import '@fontsource/inter/500.css';
import '@fontsource/inter/600.css';
import '@fontsource/inter/700.css';
import '@fontsource/inter/800.css';
import '@fontsource/inter/900.css';
import './app.css';
import App from './App.svelte';
// @ts-ignore
import { registerSW } from 'virtual:pwa-register';
import * as Sentry from '@sentry/svelte';

Sentry.init({
	dsn: import.meta.env.VITE_SENTRY_DSN,
	sendDefaultPii: false
});

registerSW({ immediate: true });

const target = document.getElementById('app');
let app;
if (target) {
	app = mount(App, { target });
}

export default app;
