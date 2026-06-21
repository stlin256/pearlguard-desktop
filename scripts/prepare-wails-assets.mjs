import fs from 'node:fs';
import path from 'node:path';

const root = process.cwd();
const out = path.join(root, 'frontend', 'dist');
const files = ['index.html', 'styles.css', 'renderer.js', 'platform-wails.js'];

fs.rmSync(out, { recursive: true, force: true });
fs.mkdirSync(path.join(out, 'assets'), { recursive: true });
for (const file of files) {
  fs.copyFileSync(path.join(root, 'src', file), path.join(out, file));
}
fs.copyFileSync(path.join(root, 'assets', 'app-icon.svg'), path.join(out, 'assets', 'app-icon.svg'));
console.log('wails assets prepared');
