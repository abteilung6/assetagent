import { useMatches } from "@tanstack/react-router";
import type React from "react";

export const PageTitle: React.FC = () => {
  const matches = useMatches();
  const title = [...matches]
    .reverse()
    .find((match) => match.staticData?.title)?.staticData?.title;

  if (!title) {
    return null;
  }

  return <h1 className="text-base font-semibold">{title}</h1>;
};
