import { act, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { testRender } from "@/test/render";

import { ThinkingIndicator } from "./thinking-indicator";

describe("ThinkingIndicator", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows an accessible thinking status with shimmer lines", () => {
    testRender(<ThinkingIndicator />);

    expect(screen.getByRole("status")).toHaveAttribute(
      "aria-label",
      "Thinking…",
    );
    expect(screen.getByText("Thinking…")).toBeInTheDocument();
    expect(screen.getByRole("status").querySelectorAll(".animate-pulse")).toHaveLength(
      3,
    );
  });

  it("cycles through status messages over time", () => {
    vi.useFakeTimers();
    testRender(<ThinkingIndicator />);

    expect(screen.getByText("Thinking…")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(2400);
    });

    expect(screen.getByText("Checking your data…")).toBeInTheDocument();
  });
});
