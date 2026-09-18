import os
import re
import json

def get_todos():
    todos = []
    # Match comments: //, /*, or # followed by 'TODO', 'FIXME', 'FIX', 'BUG', 'HACK', or 'XXX' then a space, colon or hyphen
    pattern = re.compile(r'([#/]+|/\*).*?\b(TODO|FIXME|FIX|BUG|HACK|XXX)[\s:-](.*)', re.IGNORECASE)

    extensions = {'.go': 'golang', '.js': 'javascript', '.ts': 'typescript', '.svelte': 'svelte', '.java': 'java', '.py': 'python'}
    exclude_dirs = {'node_modules', 'vendor', 'build', 'dist', 'target', '.git', 'public', 'assets', 'styles', 'e2e'}

    for root, dirs, files in os.walk('.'):
        dirs[:] = [d for d in dirs if d not in exclude_dirs]
        for file in files:
            ext = os.path.splitext(file)[1]
            if ext in extensions:
                filepath = os.path.join(root, file)
                try:
                    with open(filepath, 'r', encoding='utf-8') as f:
                        for line_num, line in enumerate(f, 1):
                            match = pattern.search(line)
                            if match:
                                marker = match.group(2).upper()
                                content = match.group(3).strip()
                                if content.endswith('*/'):
                                    content = content[:-2].strip()

                                todos.append({
                                    'file': filepath,
                                    'line': line_num,
                                    'type': marker,
                                    'text': content,
                                    'language': extensions[ext]
                                })
                except Exception:
                    pass
    return todos

if __name__ == '__main__':
    todos = get_todos()
    print(json.dumps(todos, indent=2))
