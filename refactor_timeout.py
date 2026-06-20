import os
import re

dir_path = "/Users/peterflasch/Developer/Bibliothek/api"

for root, dirs, files in os.walk(dir_path):
    for file in files:
        if file.endswith(".go"):
            filepath = os.path.join(root, file)
            with open(filepath, "r") as f:
                content = f.read()
            
            # Replace ctx, cancel := context.WithTimeout(r.Context(), <duration>) with ctx := r.Context()
            new_content = re.sub(r'ctx,\s*cancel\s*:=\s*context\.WithTimeout\(r\.Context\(\),\s*[^)]+\)', 'ctx := r.Context()', content)
            
            # Remove the subsequent `defer cancel()` which follows the ctx assignment
            new_content = re.sub(r'(ctx := r\.Context\(\))[\t ]*\r?\n[\t ]*defer cancel\(\)', r'\1', new_content)
            
            if new_content != content:
                with open(filepath, "w") as f:
                    f.write(new_content)
                print(f"Refactored {file}")
