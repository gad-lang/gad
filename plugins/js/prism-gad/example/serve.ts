// serve.ts — dev server for the demo. Bun bundles index.html (and its main.ts),
// and two routes read the repository `samples/` directory straight from the
// filesystem — the manifest is built per request at startup, and each file's
// contents are read on demand — so nothing is embedded in the bundle.
import index from "./index.html";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative, sep } from "node:path";

const SAMPLES = join(import.meta.dir, "../../../samples");
const EXTS = [".gad", ".gadt", ".gadx"];

// listSamples walks SAMPLES and returns the relative paths (with "/" separators)
// of every .gad/.gadt/.gadx file, read fresh from the filesystem.
function listSamples(dir = SAMPLES, acc: string[] = []): string[] {
  for (const name of readdirSync(dir).sort()) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) listSamples(p, acc);
    else if (EXTS.some((e) => name.endsWith(e))) acc.push(relative(SAMPLES, p).split(sep).join("/"));
  }
  return acc;
}

const server = Bun.serve({
  port: Number(process.env.PORT) || 3000,
  development: true,
  routes: {
    "/": index,
    // The tree manifest, built from the filesystem on each request.
    "/samples/manifest.json": () => Response.json(listSamples()),
    // A single sample file's contents (nested paths included), read on demand.
    "/samples/*": (req) => {
      const rel = decodeURIComponent(new URL(req.url).pathname.replace(/^\/samples\//, ""));
      if (rel.includes("..")) return new Response("bad request", { status: 400 });
      try {
        const body = readFileSync(join(SAMPLES, rel), "utf8");
        return new Response(body, { headers: { "content-type": "text/plain; charset=utf-8" } });
      } catch {
        return new Response("not found", { status: 404 });
      }
    },
  },
});

console.log(`prism-gad demo: ${server.url}`);
