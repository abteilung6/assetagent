import type React from "react";

import type { ChatToolCall } from "@/api/types.gen";

import { ToolResultCard } from "./tool-result-card";

type ToolEvidenceProps = {
  toolCalls: ChatToolCall[];
};

export const ToolEvidence: React.FC<ToolEvidenceProps> = ({ toolCalls }) => {
  if (toolCalls.length === 0) {
    return null;
  }

  return (
    <div className="mt-3 space-y-2 border-t border-border/60 pt-3">
      <p className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
        Based on your data
      </p>
      <div className="space-y-2">
        {toolCalls.map((toolCall, index) => (
          <ToolResultCard
            key={`${toolCall.name}-${index}`}
            toolCall={toolCall}
          />
        ))}
      </div>
    </div>
  );
};
