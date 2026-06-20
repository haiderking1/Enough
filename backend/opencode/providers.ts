// PORT: mirrors backend/opencode/providers.go

export const provider_opencode = "opencode-go";
export const provider_opencode_zen = "opencode-zen";
export const provider_neuralwatt = "neuralwatt";
export const provider_codex = "openai-codex";

export type provider_info = { id: string; name: string };
export type thinking_level = { id: string; name: string };
export type model_info = {
  id: string;
  name: string;
  context_window: number;
  reasoning: boolean;
  supports_images?: boolean;
  thinking_levels?: thinking_level[];
  reasoning_field?: string;
  mandatory_thinking?: boolean;
};

export const default_reasoning_levels: thinking_level[] = [];

export const model_providers = (): provider_info[] => [
  { id: provider_opencode, name: "OpenCode Go" },
  { id: provider_opencode_zen, name: "OpenCode Zen" },
  { id: provider_neuralwatt, name: "NeuralWatt" },
  { id: provider_codex, name: "OpenAI Codex" },
];

export const codex_model_order = ["gpt-5.5", "gpt-5.4-mini", "gpt-5.4", "gpt-5.3-codex", "gpt-5.3-codex-spark", "gpt-5-codex"];

export const codex_known_models: Record<string, model_info> = {
  "gpt-5.5": { id: "gpt-5.5", name: "GPT-5.5", context_window: 272_000, reasoning: true, supports_images: true },
  "gpt-5.4-mini": { id: "gpt-5.4-mini", name: "GPT-5.4 Mini", context_window: 272_000, reasoning: true, supports_images: true },
  "gpt-5.4": { id: "gpt-5.4", name: "GPT-5.4", context_window: 272_000, reasoning: true, supports_images: true },
  "gpt-5.3-codex": { id: "gpt-5.3-codex", name: "GPT-5.3 Codex", context_window: 272_000, reasoning: true, supports_images: true },
  "gpt-5.3-codex-spark": { id: "gpt-5.3-codex-spark", name: "GPT-5.3 Codex Spark", context_window: 128_000, reasoning: true, supports_images: true },
  "gpt-5-codex": { id: "gpt-5-codex", name: "GPT-5 Codex", context_window: 272_000, reasoning: true, supports_images: true },
};

import {
  normalize_model,
  sort_models,
  fallback_models,
  lookup_model
} from "./models";
import {
  catalog_model_for_provider,
  title_case_model_id
} from "./catalog";


export const codex_models = (): model_info[] => {
  const out: model_info[] = [];
  for (const id of codex_model_order) {
    const known = codex_known_models[id];
    const m = known ?? { id, name: id, reasoning: true, context_window: 272_000 };
    out.push(normalize_model({ ...m, thinking_levels: [...default_reasoning_levels] }));
  }
  return out;
};

export type registry_like = {
  codex_models_list?: () => model_info[];
  zen_models_list?: () => model_info[];
  neuralwatt_models_list?: () => model_info[];
  models?: () => model_info[];
  lookup_neuralwatt?: (id: string) => [model_info, boolean];
  lookup_codex?: (id: string) => [model_info, boolean];
};

export const models_for_provider = (provider: string, registry: registry_like | null): model_info[] => {
  switch (provider) {
    case provider_codex:
      return registry?.codex_models_list?.() ?? codex_models();
    case provider_opencode_zen: {
      const out = registry?.zen_models_list?.() ?? fallback_models(provider_opencode_zen);
      sort_models(out); return out;
    }
    case provider_neuralwatt: {
      const out = registry?.neuralwatt_models_list?.() ?? [];
      sort_models(out); return out;
    }
    default: {
      let out = registry?.models?.() ?? fallback_models(provider_opencode);
      if (out.length === 0) {
        out = fallback_models(provider_opencode);
      }
      sort_models(out);
      return out;
    }
  }
};

export const lookup_catalog_model = (id: string): [model_info, boolean] => {
  let res = lookup_model(id); if (res[1]) return res;
  res = catalog_model_for_provider(provider_opencode_zen, id); if (res[1]) return res;
  return [{ id: "", name: "", context_window: 0, reasoning: false }, false];
};

export const provider_index = (provider: string): number => {
  const providers = model_providers();
  const idx = providers.findIndex((p) => p.id === provider);
  return idx < 0 ? 0 : idx;
};

/*
PORT STATUS
source path: backend/opencode/providers.go
source lines: 145
draft lines: 111
confidence: high
status: phase_b_compile
*/
