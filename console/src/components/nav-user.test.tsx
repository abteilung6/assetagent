import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { testRender } from "@/test/render";

describe("NavUser", () => {
  it("opens menu with Account and Log out", async () => {
    const user = userEvent.setup();
    testRender({ route: "/chat" });

    const trigger = await screen.findByRole("button", { name: /Test/i });
    await user.click(trigger);

    expect(
      await screen.findByRole("menuitem", { name: /Account/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /Log out/i }),
    ).toBeInTheDocument();
  });
});
