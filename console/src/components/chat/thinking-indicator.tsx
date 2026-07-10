import type React from "react";
import { useEffect, useState } from "react";

import { cn } from "@/lib/utils";

const STATUS_MESSAGES = [
  "Thinking…",
  "Checking your data…",
  "Analyzing transactions…",
  "Preparing your answer…",
] as const;

const STATUS_INTERVAL_MS = 2400;

type ThinkingIndicatorProps = {
  className?: string;
};

export const ThinkingIndicator: React.FC<ThinkingIndicatorProps> = ({
  className,
}) => {
  const [statusIndex, setStatusIndex] = useState(0);

  useEffect(() => {
    const timer = window.setInterval(() => {
      setStatusIndex((current) => (current + 1) % STATUS_MESSAGES.length);
    }, STATUS_INTERVAL_MS);

    return () => {
      window.clearInterval(timer);
    };
  }, []);

  return (
    <div
      className={cn("space-y-3", className)}
      role="status"
      aria-live="polite"
      aria-busy="true"
      aria-label={STATUS_MESSAGES[statusIndex]}
    >
      <div className="flex items-center gap-2.5">
        <ThinkingDots />
        <p className="text-sm text-muted-foreground transition-opacity duration-300">
          {STATUS_MESSAGES[statusIndex]}
        </p>
      </div>
      <div className="space-y-2" aria-hidden>
        <ShimmerLine className="w-[min(100%,14rem)]" />
        <ShimmerLine className="w-[min(85%,11rem)] [animation-delay:150ms]" />
        <ShimmerLine className="w-[min(70%,8rem)] [animation-delay:300ms]" />
      </div>
    </div>
  );
};

const ThinkingDots: React.FC = () => {
  return (
    <span className="flex items-center gap-1" aria-hidden>
      {([0, 1, 2] as const).map((index) => (
        <span
          key={index}
          className="size-1.5 rounded-full bg-muted-foreground/70 animate-bounce"
          style={{ animationDelay: `${index * 150}ms`, animationDuration: "0.9s" }}
        />
      ))}
    </span>
  );
};

type ShimmerLineProps = {
  className?: string;
};

const ShimmerLine: React.FC<ShimmerLineProps> = ({ className }) => {
  return (
    <div
      className={cn(
        "h-2 animate-pulse rounded-full bg-foreground/10",
        className,
      )}
    />
  );
};
