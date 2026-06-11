const fs = require('fs');
const path = require('path');

function processDir(dir) {
    const files = fs.readdirSync(dir);
    for (const file of files) {
        const fullPath = path.join(dir, file);
        if (fs.statSync(fullPath).isDirectory()) {
            processDir(fullPath);
        } else if (fullPath.endsWith('.tsx') || fullPath.endsWith('.ts')) {
            let content = fs.readFileSync(fullPath, 'utf8');
            let original = content;
            
            content = content.replace(/const \[tenant(Id)?, setTenant(Id)?\] = useState\("demo-tenant"\);\n?/g, '');
            
            content = content.replace(/,\s*\{\s*headers:\s*\{\s*"x-tenant-id":\s*[a-zA-Z0-9_]+\s*\}\s*\}/g, '');
            
            content = content.replace(/\[loadAll,\s*tenant(Id)?\]/g, '[loadAll]');
            content = content.replace(/\[tenant(Id)?\]/g, '[]');
            
            // Remove tenant input UI block if present
            content = content.replace(/<div className="flex items-center gap-2">\s*<span className="text-zinc-600 text-xs">--tenant<\/span>[\s\S]*?<\/div>/g, '');

            if (content !== original) {
                fs.writeFileSync(fullPath, content);
                console.log('Updated ' + fullPath);
            }
        }
    }
}

processDir('src/app');
