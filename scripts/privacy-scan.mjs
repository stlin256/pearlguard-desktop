import fs from 'node:fs';
import path from 'node:path';

const root = process.cwd();
const ignored = new Set(['.git', 'node_modules', 'release', 'dist', 'out', 'package-lock.json']);
const patterns = [
  { name: 'Pearl address', regex: /prl1[a-z0-9]{20,}/i },
  { name: 'Windows user path', regex: /C:\\Users\\[^\\\s]+/i },
  { name: 'Wallet data subpath', regex: /wallet-data[\\/][A-Za-z0-9_-]+/i },
  { name: 'Inline password assignment', regex: /(password|passphrase)\s*[:=]\s*["'][^"']{4,}["']/i },
  { name: 'Long raw transaction-like hex', regex: /\b[a-f0-9]{48,}\b/i }
];
const findings = [];

function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (ignored.has(entry.name)) continue;
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(full);
    else scanFile(full);
  }
}

function scanFile(file) {
  const stat = fs.statSync(file);
  if (stat.size > 1024 * 1024) return;
  const text = fs.readFileSync(file, 'utf8');
  for (const pattern of patterns) {
    const match = text.match(pattern.regex);
    if (match) findings.push({ file: path.relative(root, file), type: pattern.name, match: match[0].slice(0, 80) });
  }
}

walk(root);
if (findings.length) {
  console.error(JSON.stringify(findings, null, 2));
  throw new Error(`privacy scan failed with ${findings.length} finding(s)`);
}
console.log('privacy scan ok: no configured sensitive patterns found');

