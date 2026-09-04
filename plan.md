1. Determine the actual commit changes since `main`. The `git log -1 --stat` command showed a massive diff. It seems I am on a detached head or a PR merge branch.
2. But wait, the issue is that I submitted the PR and the E2E test `m3-bauform.spec.js` failed on `/statistiken`. Wait, the E2E test failure is entirely unrelated to my Go code change.
3. Let me investigate why `/statistiken` would have an element with both border and shadow. The error says `1 Fläche(n) auf /statistiken tragen Rahmen UND Schatten zugleich.`
4. I need to find `frontend/src/routes/(app)/statistiken/+page.svelte` or the component representing `/statistiken` to remove the conflicting class.
