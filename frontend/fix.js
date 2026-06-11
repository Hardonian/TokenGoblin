const fs = require('fs');
const path = require('path');

function fixFile(filepath) {
    let content = fs.readFileSync(filepath, 'utf8');
    
    content = content.replace(/>> /g, "&gt;&gt; ");
    content = content.replace(/\[ > \]/g, "[ &gt; ]");
    content = content.replace(/\[>\]/g, "[&gt;]");
    content = content.replace(/ \{'>'\} /g, " &gt; ");
    content = content.replace(/ > /g, " &gt; ");
    
    content = content.replace(/\]">><\/span>/g, ']">&gt;&gt;</span>');
    content = content.replace(/\]">> <\/span>/g, ']">&gt;&gt; </span>');
    content = content.replace(/\]">>><\/span>/g, ']">&gt;&gt;&gt;</span>');
    content = content.replace(/\]"><\/span>/g, ']">&gt;</span>');
    
    fs.writeFileSync(filepath, content, 'utf8');
}

function walk(dir) {
    const files = fs.readdirSync(dir);
    for (const file of files) {
        const filepath = path.join(dir, file);
        if (fs.statSync(filepath).isDirectory()) {
            walk(filepath);
        } else if (filepath.endsWith('.tsx')) {
            fixFile(filepath);
        }
    }
}

walk('c:/Users/scott/GitHub/TokenGoblin/frontend/src/app');
