import { spawn } from 'node:child_process';
import electronPath from 'electron';

const electronArgs = ['.', '--self-test'];
if (process.platform === 'linux') electronArgs.push('--no-sandbox');

const child = spawn(electronPath, electronArgs, {
  stdio: ['ignore', 'pipe', 'pipe'],
  env: {
    ...process.env,
    PEARLGUARD_POOL_FIXTURE: '1',
    PEARLGUARD_DISABLE_TRANSFER: '1'
  }
});

let output = '';
let errorOutput = '';
const timer = setTimeout(() => {
  child.kill('SIGTERM');
}, 30000);

child.stdout.on('data', (chunk) => { output += chunk.toString(); });
child.stderr.on('data', (chunk) => { errorOutput += chunk.toString(); });

child.on('exit', (code) => {
  clearTimeout(timer);
  const combined = `${output}\n${errorOutput}`;
  if (code !== 0 || !combined.includes('SELF_TEST_PASS') || !combined.includes('"transferRequests":0')) {
    console.error(combined);
    process.exit(code || 1);
  }
  console.log(combined.trim());
  console.log('e2e smoke ok: Electron rendered fixture dashboard and performed no transfer operations');
});

