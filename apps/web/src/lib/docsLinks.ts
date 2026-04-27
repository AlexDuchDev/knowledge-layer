/**
 * Builds absolute URLs to markdown files in the repo docs/ folder for in-app "read more" links.
 * Set NEXT_PUBLIC_DOCS_BASE_URL to the GitHub UI base that ends with `/docs`, e.g.
 * `https://github.com/your-org/your-repo/blob/main/docs`
 */
export function docsArticleUrl(relativeDocFile: string): string | null {
  const base = process.env.NEXT_PUBLIC_DOCS_BASE_URL?.replace(/\/$/, "").trim();
  if (!base) return null;
  const path = relativeDocFile.replace(/^\//, "");
  return `${base}/${path}`;
}
