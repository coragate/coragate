import { describe, expect, it } from "vitest";
import { parseView, viewHref } from "./view";

describe("parseView", () => {
  it("sandbox、hits、settings 原样返回，其余回退到 rules", () => {
    expect(parseView("sandbox")).toBe("sandbox");
    expect(parseView("hits")).toBe("hits");
    expect(parseView("settings")).toBe("settings");
    expect(parseView(undefined)).toBe("rules");
    expect(parseView("rules")).toBe("rules");
    expect(parseView("unknown")).toBe("rules");
  });
});

describe("viewHref", () => {
  it("rules 为根路径，其它视图用 query，locale 不进路径", () => {
    expect(viewHref("rules")).toBe("/");
    expect(viewHref("sandbox")).toBe("/?view=sandbox");
    expect(viewHref("hits")).toBe("/?view=hits");
    expect(viewHref("settings")).toBe("/?view=settings");
  });
});
