// Copyright (c) 2026 Peter Flasch. All rights reserved.
// This source code is proprietary and confidential.

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
