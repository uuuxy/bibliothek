// Usage: node api.mjs METHOD PATH [json]
import { request } from '@playwright/test';
const ctx = await request.newContext({ baseURL: 'http://localhost:8085' });
const csrf = async () => (await (await ctx.get('/api/csrf-token')).json()).csrf_token;
let r = await ctx.post('/login', { headers: { 'X-CSRF-Token': await csrf() }, data: { email: 'bibliothek@musterschule.de', password: 'demo' } });
if (!r.ok()) { console.log('login', r.status(), await r.text()); process.exit(1); }
const [method, path, body] = process.argv.slice(2);
r = await ctx.fetch(path, { method, headers: { 'X-CSRF-Token': await csrf() }, data: body ? JSON.parse(body) : undefined });
console.log(r.status(), (await r.text()).slice(0, 400));
await ctx.dispose();
