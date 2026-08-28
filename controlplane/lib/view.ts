export const VIEWS = ["rules", "sandbox", "hits", "settings"] as const;

export type ViewId = (typeof VIEWS)[number];

export function parseView(raw: string | undefined | null): ViewId {
  if (raw === "sandbox" || raw === "hits" || raw === "settings") {
    return raw;
  }
  return "rules";
}

export function viewHref(view: ViewId): string {
  return view === "rules" ? "/" : `/?view=${view}`;
}
