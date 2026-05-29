import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'

const target = document.getElementById('app');
let app;
if (target) {
  app = mount(App, { target });
}

export default app
