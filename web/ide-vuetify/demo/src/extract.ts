// Archive extraction for the demo's onUpload handler: ZIP / TAR / TAR.GZ → a flat
// list of { path, content } text files. ZIP and gzip use fflate; tar is parsed
// inline (the ustar 512-byte block format).
import { gunzipSync, unzipSync } from "fflate";

export interface ExtractedFile {
  path: string;
  content: string;
}

/** base64ToBytes decodes a base64 string to a Uint8Array. */
export function base64ToBytes(b64: string): Uint8Array {
  const bin = atob(b64);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}

const dec = new TextDecoder();

/** extractArchive expands an archive's bytes into text files. Directory and empty
 * entries are skipped. */
export function extractArchive(kind: "zip" | "tar" | "tar.gz", bytes: Uint8Array): ExtractedFile[] {
  if (kind === "zip") {
    const files = unzipSync(bytes);
    return Object.entries(files)
      .filter(([name]) => !name.endsWith("/"))
      .map(([name, data]) => ({ path: name, content: dec.decode(data) }));
  }
  const tar = kind === "tar.gz" ? gunzipSync(bytes) : bytes;
  return untar(tar);
}

// untar reads the ustar/tar format: 512-byte header blocks, each followed by the
// file content padded to a 512-byte boundary. Only regular files (type '0'/'\0')
// are returned.
function untar(buf: Uint8Array): ExtractedFile[] {
  const out: ExtractedFile[] = [];
  let off = 0;
  while (off + 512 <= buf.length) {
    const header = buf.subarray(off, off + 512);
    // Two consecutive zero blocks mark the archive end.
    if (header.every((b) => b === 0)) break;
    const name = readStr(header, 0, 100);
    const size = parseInt(readStr(header, 124, 12).trim() || "0", 8) || 0;
    const type = String.fromCharCode(header[156]);
    off += 512;
    if ((type === "0" || type === "\0") && name && !name.endsWith("/")) {
      out.push({ path: name, content: dec.decode(buf.subarray(off, off + size)) });
    }
    off += Math.ceil(size / 512) * 512;
  }
  return out;
}

function readStr(block: Uint8Array, start: number, len: number): string {
  const slice = block.subarray(start, start + len);
  const end = slice.indexOf(0);
  return dec.decode(end === -1 ? slice : slice.subarray(0, end));
}
