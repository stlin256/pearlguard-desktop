import fs from 'node:fs';
import path from 'node:path';

const root = process.cwd();
const ignored = new Set(['.git', 'node_modules', 'release', 'dist', 'out']);
const jsonFiles = [];

function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (ignored.has(entry.name)) continue;
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(full);
    else if (entry.name.endsWith('.json')) jsonFiles.push(full);
  }
}

walk(root);
for (const file of jsonFiles) {
  JSON.parse(fs.readFileSync(file, 'utf8'));
}

const requiredFiles = [
  'assets/banner.svg',
  'assets/app-icon.svg',
  'src/index.html',
  'src/renderer.js',
  'src/styles.css',
  'electron/main.js',
  'electron/preload.js',
  'data/pools.example.json'
];

for (const file of requiredFiles) {
  if (!fs.existsSync(path.join(root, file))) throw new Error(`Missing required file: ${file}`);
}

console.log(`lint ok: ${jsonFiles.length} JSON files parsed, ${requiredFiles.length} required files present`);
