import { describe, expect, it } from "vitest";
import { FALLBACK_PLUGINS } from "./dataplane";
import {
  parsePlugin,
  resolvePlugin,
  toRow,
  fromRow,
  defaultActionFor,
} from "./rules-draft";

describe("parsePlugin", () => {
  it("已知 plugin 原样返回", () => {
    expect(parsePlugin("pii", FALLBACK_PLUGINS)).toBe("pii");
    expect(parsePlugin("injection", FALLBACK_PLUGINS)).toBe("injection");
    expect(parsePlugin("secret", FALLBACK_PLUGINS)).toBe("secret");
    expect(parsePlugin("keyword", FALLBACK_PLUGINS)).toBe("keyword");
  });

  it("未知 plugin 不得当成 keyword", () => {
    expect(parsePlugin("onnx", FALLBACK_PLUGINS)).toBeNull();
    expect(parsePlugin("unknown", FALLBACK_PLUGINS)).toBeNull();
  });
});

describe("toRow", () => {
  it("未知 plugin 保存时仍带原 id，不改写成 keyword", () => {
    const row = toRow(
      {
        id: "x",
        plugin: "onnx",
        type: "",
        action: "block",
        pattern: "a",
      },
      FALLBACK_PLUGINS
    );
    expect(row.plugin).toBe("onnx");
    expect(row.plugin).not.toBe("keyword");
  });

  it("pii 行写出 type，keyword 行写出 pattern", () => {
    expect(
      toRow(
        {
          id: "pii-email",
          plugin: "pii",
          type: "email",
          action: "redact",
          pattern: "",
        },
        FALLBACK_PLUGINS
      )
    ).toEqual({
      id: "pii-email",
      plugin: "pii",
      action: "redact",
      type: "email",
    });
    expect(
      toRow(
        {
          id: "demo-keyword",
          plugin: "keyword",
          type: "",
          action: "block",
          pattern: "(?i)hello",
        },
        FALLBACK_PLUGINS
      )
    ).toEqual({
      id: "demo-keyword",
      plugin: "keyword",
      action: "block",
      pattern: "(?i)hello",
    });
  });
});

describe("fromRow", () => {
  it("空 plugin 回退到目录第一项，未知 plugin 保留原值", () => {
    expect(resolvePlugin(undefined, FALLBACK_PLUGINS)).toBe("keyword");
    expect(resolvePlugin("onnx", FALLBACK_PLUGINS)).toBe("onnx");
    const unknown = fromRow({ id: "x", plugin: "onnx" }, FALLBACK_PLUGINS);
    expect(unknown.plugin).toBe("onnx");
  });

  it("缺 action 时用插件目录缺省", () => {
    expect(defaultActionFor("pii", FALLBACK_PLUGINS)).toBe("redact");
    expect(defaultActionFor("injection", FALLBACK_PLUGINS)).toBe("block");
    expect(defaultActionFor("secret", FALLBACK_PLUGINS)).toBe("block");
    expect(fromRow({ id: "p", plugin: "pii", type: "email" }, FALLBACK_PLUGINS).action).toBe(
      "redact"
    );
  });
});
