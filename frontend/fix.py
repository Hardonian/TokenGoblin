import os
import glob

def fix_file(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # We want to replace ">>" with "&gt;&gt;" if it's in text. 
    # But wait, what if it's in JS code like "x >> 1"? We don't have bitwise shifts.
    # It's safer to just replace ">>" with "&gt;&gt;" everywhere since we only used it for text.
    # Same for single ">" inside text.
    # Actually, replacing ">>" with "&gt;&gt;" is safe for our UI text.
    # Let's do replacements manually.
    content = content.replace(">> ", "&gt;&gt; ")
    content = content.replace("[ > ]", "[ &gt; ]")
    content = content.replace("[>]", "[&gt;]")
    content = content.replace(" {'>'} ", " &gt; ")
    content = content.replace(" > ", " &gt; ")
    
    # We also have <span className="text-[#ffb000]">></span>
    content = content.replace(']">></span>', ']">&gt;&gt;</span>')
    content = content.replace(']">> </span>', ']">&gt;&gt; </span>')
    content = content.replace(']">>></span>', ']">&gt;&gt;&gt;</span>')
    content = content.replace(']">></span>', ']">&gt;&gt;</span>')
    content = content.replace(']"></span>', ']">&gt;</span>')
    
    with open(filepath, 'w', encoding='utf-8') as f:
        f.write(content)

for root, _, files in os.walk('c:/Users/scott/GitHub/TokenGoblin/frontend/src/app'):
    for file in files:
        if file.endswith('.tsx'):
            fix_file(os.path.join(root, file))
