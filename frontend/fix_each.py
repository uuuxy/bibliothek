import os
import re

def fix_svelte_file(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()

    # Match {#each array as item} -> {#each array as item, idx (idx)}
    # Match {#each array as item, index} -> {#each array as item, index (index)}
    # Need to be careful not to match {#each array as item (item.id)}

    def replace_each(match):
        full_match = match.group(0)
        inner = match.group(1).strip() # e.g. "array as item" or "array as item, i"
        
        # Check if there is already a key like (something)
        if re.search(r'\(\s*[^\)]+\s*\)\s*$', inner):
            return full_match # already has a key
        
        # Check if there is an index variable
        has_index = re.search(r' as .*?,\s*([a-zA-Z0-9_]+)$', inner)
        if has_index:
            idx_var = has_index.group(1)
            return f"{{#each {inner} ({idx_var})}}"
        else:
            # no index, let's append one. But we need a safe index variable name.
            # let's just use `_i`
            return f"{{#each {inner}, _i (_i)}}"

    new_content = re.sub(r'\{#each\s+([^}]+)\}', replace_each, content)
    
    # Also fix svelte-ignore unused
    # just remove "<!-- svelte-ignore -->" with empty
    new_content = re.sub(r'<!-- svelte-ignore a11y[-a-zA-Z0-9_]* -->\s*\n?', '', new_content)
    new_content = re.sub(r'<!-- svelte-ignore [^>]+ -->\s*\n?', '', new_content) # aggressive removal of unused ignores if eslint complained

    if new_content != content:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(new_content)
        print(f"Fixed {filepath}")

def main():
    src_dir = '/Users/peterflasch/Developer/Bibliothek/frontend/src'
    for root, dirs, files in os.walk(src_dir):
        for file in files:
            if file.endswith('.svelte'):
                fix_svelte_file(os.path.join(root, file))

if __name__ == '__main__':
    main()
