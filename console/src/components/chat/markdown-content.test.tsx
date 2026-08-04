import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MarkdownContent } from "./markdown-content";

describe("MarkdownContent", () => {
  it("renders bold, headings, and lists from markdown", () => {
    render(
      <MarkdownContent
        content={`### Money Review Highlights:
- **Free Cashflow:** €840.11/month
- **Finding 1:** amount changed`}
      />,
    );

    expect(
      screen.getByRole("heading", { name: /Money Review Highlights/i }),
    ).toBeInTheDocument();
    expect(screen.getByText("Free Cashflow:")).toBeInTheDocument();
    expect(screen.getByText(/€840.11\/month/)).toBeInTheDocument();
    expect(screen.queryByText(/\*\*Free Cashflow:\*\*/)).not.toBeInTheDocument();
  });
});
