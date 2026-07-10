import type React from "react";
import { useState } from "react";
import { ArrowUpIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type ComposerProps = {
  onSend: (content: string) => void | Promise<void>;
  disabled?: boolean;
};

export const Composer: React.FC<ComposerProps> = ({
  onSend,
  disabled = false,
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

  return (
    <div className="shrink-0 border-t bg-background px-4 py-4">
      <div className="mx-auto flex w-full max-w-3xl items-end gap-2 rounded-2xl border bg-muted/30 p-2">
        <textarea
          value={value}
          onChange={(event) => setValue(event.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Ask about your finances…"
          rows={1}
          disabled={disabled}
          className={cn(
            "max-h-40 min-h-10 flex-1 resize-none bg-transparent px-2 py-2 text-sm outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50",
          )}
        />
        <Button
          type="button"
          size="icon"
          onClick={() => void handleSubmit()}
          disabled={disabled || value.trim().length === 0}
          aria-label="Send message"
        >
          <ArrowUpIcon />
        </Button>
      </div>
    </div>
  );
};
