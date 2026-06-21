import fs from 'node:fs';
import path from 'node:path';
import zlib from 'node:zlib';

const size = 256;
const scale = size / 128;
const pixels = Buffer.alloc(size * size * 4);
const rgbaPixels = Buffer.alloc(size * size * 4);

function hex(color) {
  const value = color.replace('#', '');
  return {
    r: Number.parseInt(value.slice(0, 2), 16),
    g: Number.parseInt(value.slice(2, 4), 16),
    b: Number.parseInt(value.slice(4, 6), 16),
    a: value.length >= 8 ? Number.parseInt(value.slice(6, 8), 16) : 255
  };
}

function setPixel(x, y, color) {
  if (x < 0 || y < 0 || x >= size || y >= size) return;
  const index = (y * size + x) * 4;
  pixels[index] = color.b;
  pixels[index + 1] = color.g;
  pixels[index + 2] = color.r;
  pixels[index + 3] = color.a;
  rgbaPixels[index] = color.r;
  rgbaPixels[index + 1] = color.g;
  rgbaPixels[index + 2] = color.b;
  rgbaPixels[index + 3] = color.a;
}

function roundedRect(x, y, w, h, radius, color) {
  x = Math.round(x * scale);
  y = Math.round(y * scale);
  w = Math.round(w * scale);
  h = Math.round(h * scale);
  radius = Math.round(radius * scale);
  for (let py = y; py < y + h; py += 1) {
    for (let px = x; px < x + w; px += 1) {
      const dx = px < x + radius ? x + radius - px : px >= x + w - radius ? px - (x + w - radius - 1) : 0;
      const dy = py < y + radius ? y + radius - py : py >= y + h - radius ? py - (y + h - radius - 1) : 0;
      if (dx * dx + dy * dy <= radius * radius) setPixel(px, py, color);
    }
  }
}

function circle(cx, cy, radius, color) {
  cx *= scale;
  cy *= scale;
  radius *= scale;
  for (let y = Math.floor(cy - radius); y <= Math.ceil(cy + radius); y += 1) {
    for (let x = Math.floor(cx - radius); x <= Math.ceil(cx + radius); x += 1) {
      const dx = x - cx;
      const dy = y - cy;
      if (dx * dx + dy * dy <= radius * radius) setPixel(x, y, color);
    }
  }
}

function line(x1, y1, x2, y2, width, color) {
  const steps = Math.max(Math.abs(x2 - x1), Math.abs(y2 - y1)) * scale;
  for (let i = 0; i <= steps; i += 1) {
    const t = steps === 0 ? 0 : i / steps;
    circle(x1 + (x2 - x1) * t, y1 + (y2 - y1) * t, width / 2, color);
  }
}

const pearl = hex('#f7fbfb');
const shell = hex('#eef6f4');
const teal = hex('#138f8f');
const navy = hex('#17324d');
const violet = hex('#7568c9');
const white = hex('#ffffff');

roundedRect(2, 2, 124, 124, 28, pearl);
line(64, 16, 106, 31, 7, teal);
line(106, 31, 106, 63, 7, teal);
line(106, 63, 64, 112, 7, teal);
line(64, 112, 22, 63, 7, teal);
line(22, 63, 22, 31, 7, teal);
line(22, 31, 64, 16, 7, teal);
roundedRect(33, 34, 62, 65, 20, shell);
circle(64, 63, 27, white);
circle(64, 63, 22, hex('#f7fbfb'));
line(48, 66, 56, 55, 8, teal);
line(56, 55, 72, 55, 8, teal);
line(72, 55, 80, 66, 8, teal);
line(45, 88, 83, 88, 8, navy);
circle(64, 63, 27, violet);
circle(64, 63, 21, white);
line(49, 66, 58, 56, 8, teal);
line(58, 56, 70, 56, 8, teal);
line(70, 56, 79, 66, 8, teal);

const dibHeaderSize = 40;
const pixelBytes = size * size * 4;
const maskRowBytes = Math.ceil(size / 32) * 4;
const maskBytes = maskRowBytes * size;
const imageBytes = dibHeaderSize + pixelBytes + maskBytes;
const ico = Buffer.alloc(6 + 16 + imageBytes);
let offset = 0;
ico.writeUInt16LE(0, offset); offset += 2;
ico.writeUInt16LE(1, offset); offset += 2;
ico.writeUInt16LE(1, offset); offset += 2;
ico.writeUInt8(size === 256 ? 0 : size, offset); offset += 1;
ico.writeUInt8(size === 256 ? 0 : size, offset); offset += 1;
ico.writeUInt8(0, offset); offset += 1;
ico.writeUInt8(0, offset); offset += 1;
ico.writeUInt16LE(1, offset); offset += 2;
ico.writeUInt16LE(32, offset); offset += 2;
ico.writeUInt32LE(imageBytes, offset); offset += 4;
ico.writeUInt32LE(22, offset); offset += 4;
ico.writeUInt32LE(dibHeaderSize, offset); offset += 4;
ico.writeInt32LE(size, offset); offset += 4;
ico.writeInt32LE(size * 2, offset); offset += 4;
ico.writeUInt16LE(1, offset); offset += 2;
ico.writeUInt16LE(32, offset); offset += 2;
ico.writeUInt32LE(0, offset); offset += 4;
ico.writeUInt32LE(pixelBytes, offset); offset += 4;
ico.writeInt32LE(0, offset); offset += 4;
ico.writeInt32LE(0, offset); offset += 4;
ico.writeUInt32LE(0, offset); offset += 4;
ico.writeUInt32LE(0, offset); offset += 4;

for (let y = size - 1; y >= 0; y -= 1) {
  pixels.copy(ico, offset, y * size * 4, (y + 1) * size * 4);
  offset += size * 4;
}
// AND mask remains zero.

fs.writeFileSync(path.join(process.cwd(), 'assets', 'icon.ico'), ico);
writePng(path.join(process.cwd(), 'assets', 'icon.png'));
console.log('generated assets/icon.ico and assets/icon.png');

function crc32(buffer) {
  let crc = 0xffffffff;
  for (const byte of buffer) {
    crc ^= byte;
    for (let bit = 0; bit < 8; bit += 1) {
      crc = (crc >>> 1) ^ (0xedb88320 & -(crc & 1));
    }
  }
  return (crc ^ 0xffffffff) >>> 0;
}

function pngChunk(type, data) {
  const typeBuffer = Buffer.from(type, 'ascii');
  const chunk = Buffer.alloc(8 + data.length + 4);
  chunk.writeUInt32BE(data.length, 0);
  typeBuffer.copy(chunk, 4);
  data.copy(chunk, 8);
  chunk.writeUInt32BE(crc32(Buffer.concat([typeBuffer, data])), 8 + data.length);
  return chunk;
}

function writePng(filePath) {
  const signature = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(size, 0);
  ihdr.writeUInt32BE(size, 4);
  ihdr.writeUInt8(8, 8);
  ihdr.writeUInt8(6, 9);
  ihdr.writeUInt8(0, 10);
  ihdr.writeUInt8(0, 11);
  ihdr.writeUInt8(0, 12);
  const scanline = Buffer.alloc((size * 4 + 1) * size);
  for (let y = 0; y < size; y += 1) {
    const rowOffset = y * (size * 4 + 1);
    scanline[rowOffset] = 0;
    rgbaPixels.copy(scanline, rowOffset + 1, y * size * 4, (y + 1) * size * 4);
  }
  const idat = zlib.deflateSync(scanline, { level: 9 });
  fs.writeFileSync(filePath, Buffer.concat([
    signature,
    pngChunk('IHDR', ihdr),
    pngChunk('IDAT', idat),
    pngChunk('IEND', Buffer.alloc(0))
  ]));
}
