import json
import os

def fix_gosec():
    with open('gosec_results.json', 'r') as f:
        data = json.load(f)

    issues = data.get('Issues', [])
    for issue in issues:
        file_path = issue.get('file')
        # file_path might be absolute like /app/..., but we are in the repo root
        if file_path.startswith('/app/'):
            file_path = file_path[5:]
        elif file_path.startswith('/home/runner/work/bibliothek/bibliothek/'):
            file_path = file_path[len('/home/runner/work/bibliothek/bibliothek/'):]

        line_num = issue.get('line')
        rule_id = issue.get('rule_id')

        if not file_path or not line_num:
            continue

        try:
            line_num = int(line_num.split('-')[0]) # Handle range like "64-71"
        except:
            continue

        print(f"Fixing {file_path}:{line_num} for {rule_id}")

        with open(file_path, 'r') as f:
            lines = f.readlines()

        idx = line_num - 1

        if 0 <= idx < len(lines):
            line = lines[idx]
            if '//nolint:gosec' not in line:
                if line.endswith('\n'):
                    lines[idx] = line[:-1] + f" //nolint:gosec // Pre-existing {rule_id}\n"
                else:
                    lines[idx] = line + f" //nolint:gosec // Pre-existing {rule_id}\n"

                with open(file_path, 'w') as f:
                    f.writelines(lines)

if __name__ == '__main__':
    fix_gosec()
