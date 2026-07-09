import type React from "react";

import { ThemeToggle } from "@/components/theme-toggle";

const App: React.FC = () => {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-4">
      <ThemeToggle />
      <h1 className="text-2xl font-semibold">assetagent console</h1>
    </main>
  );
};

export default App;
