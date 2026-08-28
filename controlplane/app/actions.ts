"use server";

import { revalidatePath } from "next/cache";
import { inspectText, putRules, type InspectResult } from "@/lib/dataplane";

export type ActionState = {
  ok: boolean;
  message: string;
  inspect?: InspectResult;
};

export async function saveRulesAction(
  _prev: ActionState,
  formData: FormData
): Promise<ActionState> {
  const raw = String(formData.get("snapshot") ?? "");
  let snap: {
    schema_version: number;
    rules: {
      id: string;
      plugin?: string;
      pattern?: string;
      type?: string;
      action?: string;
    }[];
  };
  try {
    snap = JSON.parse(raw) as typeof snap;
  } catch {
    return { ok: false, message: "规则 JSON 无法解析" };
  }
  try {
    await putRules({
      schema_version: snap.schema_version || 1,
      rules: (snap.rules ?? []).map((r) => ({
        id: r.id,
        plugin: r.plugin,
        pattern: r.pattern,
        type: r.type,
        action: r.action,
      })),
    });
    revalidatePath("/");
    return { ok: true, message: "已写入数据面规则快照" };
  } catch (e) {
    return { ok: false, message: e instanceof Error ? e.message : "保存失败" };
  }
}

export async function inspectAction(
  _prev: ActionState,
  formData: FormData
): Promise<ActionState> {
  const text = String(formData.get("text") ?? "");
  if (!text.trim()) {
    return { ok: false, message: "请粘贴要检测的文本" };
  }
  try {
    const inspect = await inspectText(text);
    return { ok: true, message: inspect.engine_error ? "检测引擎不可用" : "已调用数据面检测端点", inspect };
  } catch (e) {
    return { ok: false, message: e instanceof Error ? e.message : "检测失败" };
  }
}
