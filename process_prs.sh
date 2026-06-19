#!/bin/bash
set -e

cd /Users/peterflasch/Developer/Bibliothek

echo "Fetching open PRs..."
prs=$(gh pr list --state open --json number -q '.[].number')

echo "Found PRs:"
echo "$prs"

for pr in $prs; do
    echo "========================================="
    echo "Processing PR #$pr"
    
    # Try to check out the PR
    if ! gh pr checkout $pr; then
        echo "Failed to checkout PR #$pr. Closing it."
        gh pr close $pr -c "Automatisch geschlossen: Konnte PR nicht auschecken."
        git checkout main
        continue
    fi
    
    # Rebase onto main to get test fixes
    echo "Rebasing PR #$pr onto main..."
    if ! git rebase main; then
        echo "Rebase failed for PR #$pr. Aborting and closing."
        git rebase --abort
        gh pr close $pr -c "Automatisch geschlossen: Merge-/Rebase-Konflikt mit main."
        git checkout main
        continue
    fi
    
    echo "Running tests for PR #$pr..."
    if GOWORK=off go build ./... && GOWORK=off go test ./...; then
        echo "Tests passed! Merging PR #$pr..."
        git checkout main
        gh pr merge $pr --squash --delete-branch || echo "Failed to merge PR #$pr"
    else
        echo "Tests failed! Closing PR #$pr..."
        git checkout main
        gh pr close $pr -c "Automatisch geschlossen: Build oder Tests fehlgeschlagen."
    fi
    
    # Ensure we are back on main and fully updated for the next PR
    git checkout main
    git pull origin main --rebase
    
    # Cleanup local branch if it was created
    branch_name=$(gh pr view $pr --json headRefName -q .headRefName)
    if [ -n "$branch_name" ]; then
        git branch -D "$branch_name" || true
    fi
done

echo "========================================="
echo "All PRs processed!"
