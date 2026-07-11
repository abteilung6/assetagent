import { toolDisplayName } from "@/components/chat/tool-sources";
import { cn } from "@/lib/utils";

type ToolCallIndicatorProps = {
  toolName: string;
  className?: string;
};

export const ToolCallIndicator: React.FC<ToolCallIndicatorProps> = ({
  toolName,
  className,
}) => {
  return (
    <p
      className={cn("text-sm text-muted-foreground", className)}
      role="status"
      aria-live="polite"
    >
      Running {toolDisplayName(toolName)}…
    </p>
  );
};
