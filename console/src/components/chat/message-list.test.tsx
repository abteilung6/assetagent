import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { MessageList } from "./message-list";
import { defaultStarters } from "./starters";

describe("MessageList empty state", () => {
  it("renders four starters and sends on click", async () => {
    const user = userEvent.setup();
    const onStarter = vi.fn();
    const starters = defaultStarters(new Date(2026, 6, 19));

    render(
      <MessageList
        messages={[]}
        starters={starters}
        onStarter={onStarter}
      />,
    );

    expect(
      screen.getByText(/trusted cashflow and Baseline tools/i),
    ).toBeInTheDocument();

    const buttons = screen.getAllByRole("button");
    expect(buttons).toHaveLength(4);

    await user.click(buttons[0]!);
    expect(onStarter).toHaveBeenCalledWith(starters[0]!.prompt);
  });
});
