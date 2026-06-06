#!/bin/bash

# Ensure we are on main and clean
git checkout main

# Close all open PRs
open_prs=$(gh pr list --state open --json number --jq '.[].number')
for pr in $open_prs; do
    echo "Closing PR #$pr"
    gh pr close "$pr" -c "Closed during master repository cleanup."
done

# Get all remote branches except main, HEAD, and our survivor
branches=$(git branch -r | grep -v 'origin/main' | grep -v 'origin/HEAD' | grep -v 'origin/fix/barcode-documentation-7150129749818138239' | sed 's/origin\///' | tr -d ' ')

for b in $branches; do
    if [ -n "$b" ]; then
        echo "Deleting branch: $b"
        # Delete remote
        git push origin --delete "$b" || true
        # Delete local
        git branch -D "$b" || true
    fi
done

echo "Done deleting branches."
