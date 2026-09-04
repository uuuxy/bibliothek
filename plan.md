1. Use `replace_with_git_merge_diff` to introduce a `FeldKontext` struct in `internal/uebernahme/felder.go` to group `QuellID`, `Kennung`, `Feld`, `Wert` and `Max`. Update `Kuerze` and `KuerzeNullbar` to take `FeldKontext` instead of individual string parameters.
2. Use `replace_with_git_merge_diff` to update `cmd/migrate/pg_writer.go` and replace `uebernahme.Kuerze(el, id, isbn, "titel", m.Titel, maxTitelSpalte)` with `uebernahme.Kuerze(el, uebernahme.FeldKontext{QuellID: id, Kennung: isbn, Feld: "titel", Wert: m.Titel, Max: maxTitelSpalte})` and similarly for `KuerzeNullbar`.
3. Use `replace_with_git_merge_diff` to update `internal/littera/schreiber_bestand.go` to pass `uebernahme.FeldKontext` in the closure helper functions `k` and `kn`.
4. Use `replace_with_git_merge_diff` to update `internal/littera/schreiber_personen.go` to pass `uebernahme.FeldKontext` in `p.kuerze`.
5. Use `replace_with_git_merge_diff` to update `internal/uebernahme/felder_test.go` and `internal/uebernahme/uebernahme_test.go` tests to match the new signature.
6. Use `replace_with_git_merge_diff` to update `internal/littera/schreiber_personen_test.go` tests if necessary.
7. Use `run_in_bash_session` to compile and run tests (`go test ./...`) to verify changes.
8. Use `pre_commit_instructions` tool to run the necessary pre-commit hooks to verify changes.
9. Use `submit` to create a PR with the code health improvements.
