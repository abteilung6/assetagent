import type React from "react";

import { useHealth } from "@/hooks/use-health";

export const HealthStatus: React.FC = () => {
  const { data, isError, isPending } = useHealth();

  if (isPending) {
    return (
      <span
        className="text-xs text-muted-foreground"
        data-testid="health-status"
      >
        API…
      </span>
    );
  }

  if (isError) {
    return (
      <span className="text-xs text-destructive" data-testid="health-status">
        API offline
      </span>
    );
  }

  return (
    <span
      className="text-xs text-muted-foreground"
      data-testid="health-status"
    >
      API {data.status}
    </span>
  );
};
