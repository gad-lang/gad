// WebFS — a read-only bundled sample tree with a LocalStorage overlay for user
// changes, for the embeddable in-browser IDE. The bundled `samples` are never
// mutated (editing one stores an override in LocalStorage); the user may also
// create files/directories and delete anything. A reset clears the overlay,
// returning to the pristine samples.
import { samples } from "./samples.gen";

const LS_KEY = "gad-webide-fs-v1";

interface Overlay {
  // Path -> content for edited samples and user-created files.
  files: Record<string, string>;
  // Explicitly created (possibly empty) directories, as "dir/".
  dirs: string[];
  // Paths hidden from the base tree (deleted samples) or deleted user paths.
  deleted: string[];
}

function emptyOverlay(): Overlay {
  return { files: {}, dirs: [], deleted: [] };
}

export class WebFS {
  private overlay: Overlay;

  constructor() {
    this.overlay = this.load();
  }

  private load(): Overlay {
    try {
      const raw = localStorage.getItem(LS_KEY);
      if (raw) return { ...emptyOverlay(), ...(JSON.parse(raw) as Overlay) };
    } catch {
      /* ignore corrupt overlay */
    }
    return emptyOverlay();
  }

  private save(): void {
    try {
      localStorage.setItem(LS_KEY, JSON.stringify(this.overlay));
    } catch {
      /* storage full / unavailable — changes stay in memory this session */
    }
  }

  private isDeleted(path: string): boolean {
    return this.overlay.deleted.includes(path) ||
      // A path under a deleted directory is also gone.
      this.overlay.deleted.some((d) => d.endsWith("/") && path.startsWith(d));
  }

  /** list returns every visible file path (samples minus deletions, plus user
   * files), sorted. Directories are implied by the "/" in paths and by dirs. */
  listFiles(): string[] {
    const set = new Set<string>();
    for (const p of Object.keys(samples)) if (!this.isDeleted(p)) set.add(p);
    for (const p of Object.keys(this.overlay.files)) if (!this.isDeleted(p)) set.add(p);
    return [...set].sort();
  }

  /** listDirs returns explicitly-created (possibly empty) directories. */
  listDirs(): string[] {
    return this.overlay.dirs.filter((d) => !this.isDeleted(d)).sort();
  }

  exists(path: string): boolean {
    return this.listFiles().includes(path);
  }

  /** read returns a file's content (the overlay wins over the base sample). */
  read(path: string): string | undefined {
    if (this.isDeleted(path)) return undefined;
    if (path in this.overlay.files) return this.overlay.files[path];
    if (path in samples) return samples[path];
    return undefined;
  }

  /** readOnlyBase reports whether path is a pristine sample (no user override). */
  readOnlyBase(path: string): boolean {
    return path in samples && !(path in this.overlay.files);
  }

  /** write stores content for path in the overlay (editing a sample or a user
   * file). Un-deletes the path if it was deleted. */
  write(path: string, content: string): void {
    this.overlay.files[path] = content;
    this.overlay.deleted = this.overlay.deleted.filter((d) => d !== path);
    this.save();
  }

  /** createFile makes a new (empty) file. Returns false if it already exists. */
  createFile(path: string): boolean {
    if (this.exists(path)) return false;
    this.write(path, "");
    return true;
  }

  /** createDir records an (empty) directory (stored as "path/"). */
  createDir(path: string): boolean {
    const dir = path.endsWith("/") ? path : path + "/";
    if (this.overlay.dirs.includes(dir)) return false;
    this.overlay.dirs.push(dir);
    this.overlay.deleted = this.overlay.deleted.filter((d) => d !== dir);
    this.save();
    return true;
  }

  /** remove deletes a file or a directory (and everything under it). */
  remove(path: string): void {
    const isDir = path.endsWith("/");
    // Drop overlay entries at/under the path.
    for (const p of Object.keys(this.overlay.files)) {
      if (p === path || (isDir && p.startsWith(path))) delete this.overlay.files[p];
    }
    this.overlay.dirs = this.overlay.dirs.filter((d) => d !== path && !(isDir && d.startsWith(path)));
    // Hide base samples at/under the path.
    const hidesBase = path in samples || Object.keys(samples).some((p) => isDir && p.startsWith(path));
    if (hidesBase && !this.overlay.deleted.includes(path)) this.overlay.deleted.push(path);
    this.save();
  }

  /** reset clears all user changes, returning to the pristine samples. */
  reset(): void {
    this.overlay = emptyOverlay();
    try {
      localStorage.removeItem(LS_KEY);
    } catch {
      /* ignore */
    }
  }

  /** hasChanges reports whether the overlay holds any user modification. */
  hasChanges(): boolean {
    return (
      Object.keys(this.overlay.files).length > 0 ||
      this.overlay.dirs.length > 0 ||
      this.overlay.deleted.length > 0
    );
  }
}
