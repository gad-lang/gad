// Content model emitted by cmd/build-website into <base>/content.json.

export interface TocEntry {
  id: string;
  text: string;
  level: number;
}

export interface DocPage {
  slug: string;
  title: string;
  html: string;
  toc: TocEntry[];
  source?: string;
  sourceLang?: string;
  sourcePath?: string;
}

export interface NavPage {
  slug?: string;
  title: string;
  href?: string;
  // children is set for a submenu entry: a subdirectory of pages with no page of
  // its own, nested recursively.
  children?: NavPage[];
}

export interface NavGroup {
  name: string;
  pages: NavPage[];
}

export interface SearchDoc {
  slug: string;
  title: string;
  text: string;
}

export interface SiteInfo {
  repoURL: string;
  commit: string;
  playHref: string;
  downloadSlug: string;
  releaseName: string;
  hasRelease: boolean;
  tagline: string;
}

export interface SiteContent {
  site: SiteInfo;
  home: string;
  groups: NavGroup[];
  pages: Record<string, DocPage>;
  search: SearchDoc[];
}

// appBase returns the directory that serves index.html (a version dir such as
// /latest/ on the deployed site, or / locally). Captured in index.html before
// any SPA-redirect rewrite.
export function appBase(): string {
  const b = (window as unknown as { __BASE__?: string }).__BASE__ || location.pathname;
  return b.endsWith("/") ? b : b.replace(/[^/]*$/, "");
}

let cache: Promise<SiteContent> | null = null;

export function loadContent(): Promise<SiteContent> {
  if (!cache) {
    cache = fetch(appBase() + "content.json", { cache: "no-cache" }).then((r) => {
      if (!r.ok) throw new Error("content.json " + r.status);
      return r.json() as Promise<SiteContent>;
    });
  }
  return cache;
}
