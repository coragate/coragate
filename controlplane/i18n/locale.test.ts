import { describe, expect, it } from "vitest";
import { resolveLocale } from "./locale";

describe("resolveLocale：无 cookie 时的回退", () => {
  it("无 cookie 且无 Accept-Language 时默认为 en", () => {
    expect(resolveLocale(undefined, undefined)).toBe("en");
    expect(resolveLocale("", "")).toBe("en");
  });

  it("Accept-Language 含 zh 时选 zh-CN", () => {
    expect(resolveLocale(undefined, "zh-CN,zh;q=0.9,en;q=0.8")).toBe("zh-CN");
    expect(resolveLocale(undefined, "zh")).toBe("zh-CN");
  });

  it("Accept-Language 仅 en 时选 en", () => {
    expect(resolveLocale(undefined, "en-US,en;q=0.9")).toBe("en");
  });
});

describe("resolveLocale：cookie 优先", () => {
  it("有效 cookie 压过 Accept-Language", () => {
    expect(resolveLocale("en", "zh-CN")).toBe("en");
    expect(resolveLocale("zh-CN", "en-US")).toBe("zh-CN");
  });

  it("未知 cookie 不 404，按 AC-1 回退", () => {
    expect(resolveLocale("fr", undefined)).toBe("en");
    expect(resolveLocale("fr", "zh-CN")).toBe("zh-CN");
  });
});
