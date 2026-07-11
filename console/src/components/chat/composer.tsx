import type React from "react";
import { useState } from "react";
import { ArrowUpIcon } from "lucide-react";

import type { LlmModelOption, LlmModelSelection } from "@/api/types.gen";
import { ModelSelect } from "@/components/chat/model-select";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type ComposerProps = {
  onSend: (content: string) => void | Promise<void>;
  disabled?: boolean;
  modelOptions?: LlmModelOption[];
  modelSelection?: LlmModelSelection | null;
  onModelChange?: (selection: LlmModelSelection) => void;
  showModelSelect?: boolean;
};

export const Composer: React.FC<ComposerProps> = ({
  onSend,
  disabled = false,
  modelOptions = [],
  modelSelection = null,
  onModelChange,
  showModelSelect = false,
}) => {
  const [value, setValue] = useState("");

  const handleSubmit = async () => {
    const trimmed = value.trim();
    if (!trimmed || disabled) {
      return;
    }

    setValue("");
    await onSend(trimmed);
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      void handleSubmit();
    }
  };

  const canPickModel =
    showModelSelect &&
    modelSelection &&
    onModelChange &&
    modelOptions.length > 1;

  return (
    <div className="shrink-0 border-t bg-background px-3 py-3 sm:px-4 sm:py-4">
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-1 rounded-2xl border bg-muted/30 p-2 sm:flex-row sm:items-end sm:gap-2">
        <textarea
          value={value}
          onChange={(event) => setValue(event.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Ask about your finances…"
          rows={1}
          disabled={disabled}
          className={cn(
            "max-h-40 min-h-10 w-full min-w-0 flex-1 resize-none bg-transparent px-2 py-2 text-sm outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50",
          )}
        />
        <div
          className={cn(
            "flex items-center gap-1 px-0.5 sm:shrink-0 sm:pb-0.5",
            canPickModel ? "justify-between sm:justify-start" : "justify-end",
          )}
        >
          {canPickModel ? (
            <ModelSelect
              options={modelOptions}
              value={modelSelection}
              onChange={onModelChange}
              disabled={disabled}
            />
          ) : null}
          <Button
            type="button"
            size="icon"
            onClick={() => void handleSubmit()}
            disabled={disabled || value.trim().length === 0}
            aria-label="Send message"
            className="size-8 shrink-0 rounded-full"
          >
            <ArrowUpIcon />
          </Button>
        </div>
      </div>
    </div>
  );
};
