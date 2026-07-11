import { useQuery } from "@tanstack/react-query";
import { useCallback, useMemo, useState } from "react";

import { getLlmModelsOptions } from "@/api/@tanstack/react-query.gen";
import type { LlmModelCatalog, LlmModelOption, LlmModelSelection } from "@/api/types.gen";

const STORAGE_KEY = "assetagent.chat.llm";

function readStoredSelection(): LlmModelSelection | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw) as LlmModelSelection;
    if (
      parsed &&
      (parsed.provider === "ollama" || parsed.provider === "openrouter") &&
      typeof parsed.model === "string" &&
      parsed.model.length > 0
    ) {
      return parsed;
    }
  } catch {
    return null;
  }
  return null;
}

function findOption(
  options: LlmModelOption[],
  selection: LlmModelSelection,
): LlmModelOption | undefined {
  return options.find(
    (option) =>
      option.provider === selection.provider && option.model === selection.model,
  );
}

function resolveSelection(
  catalog: LlmModelCatalog,
  preferred: LlmModelSelection | null,
): LlmModelSelection | null {
  if (catalog.options.length === 0) {
    return null;
  }

  if (preferred && findOption(catalog.options, preferred)) {
    return preferred;
  }

  if (findOption(catalog.options, catalog.default)) {
    return catalog.default;
  }

  const first = catalog.options[0];
  return { provider: first.provider, model: first.model };
}

export function useLLMModels() {
  const query = useQuery({
    ...getLlmModelsOptions(),
    staleTime: 5 * 60_000,
  });
  const [preferred, setPreferred] = useState<LlmModelSelection | null>(
    readStoredSelection,
  );

  const selection = useMemo(() => {
    if (!query.data) {
      return null;
    }
    return resolveSelection(query.data, preferred);
  }, [query.data, preferred]);

  const setModel = useCallback((next: LlmModelSelection) => {
    setPreferred(next);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
  }, []);

  const options = query.data?.options ?? [];
  const canChat = selection !== null && options.length > 0;

  return {
    catalog: query.data,
    options,
    selection,
    setModel,
    canChat,
    isLoading: query.isLoading,
    isError: query.isError,
    showModelSelect: options.length > 1,
  };
}
