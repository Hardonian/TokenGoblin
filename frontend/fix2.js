const fs = require('fs');
const path = require('path');

function fixFile(filepath) {
    let content = fs.readFileSync(filepath, 'utf8');
    
    // First, fix the logical conditions
    content = content.replace(/heatmap\.heatmap_cells\.length &gt; 0/g, "heatmap.heatmap_cells.length > 0");
    content = content.replace(/m\.acceptance_rate &gt; 0\.8/g, "m.acceptance_rate > 0.8");
    content = content.replace(/m\.acceptance_rate &gt; 0\.5/g, "m.acceptance_rate > 0.5");

    // Replace the &gt;&gt; strings with {'>>'}
    content = content.replace(/&gt;&gt;/g, "{'>>'}");
    // Replace the &gt; strings with {'>'}
    content = content.replace(/&gt;/g, "{'>'}");

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
