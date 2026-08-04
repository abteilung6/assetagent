import type React from "react";
import { Navigate } from "@tanstack/react-router";

import { useMe } from "@/hooks/use-me";

export const AuthGate: React.FC<React.PropsWithChildren> = ({ children }) => {
  const me = useMe();

  if (me.isPending) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-muted/40">
        <p className="text-sm text-muted-foreground">Sitzung wird geprüft…</p>
      </div>
    );
  }

  if (me.isError || !me.data) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
};
