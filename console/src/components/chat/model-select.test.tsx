import type React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { ModelSelect } from "@/components/chat/model-select";
import { TooltipProvider } from "@/components/ui/tooltip";
import { testRender } from "@/test/render";

function renderModelSelect(ui: React.ReactElement) {
  return render(<TooltipProvider>{ui}</TooltipProvider>);
}

describe("ModelSelect", () => {
  it("is hidden when only one model is available", () => {
    const { container } = renderModelSelect(
      <ModelSelect
        options={[
          {
            provider: "openrouter",
            model: "openai/gpt-4o-mini",
            label: "GPT-4o mini",
          },
        ]}
        value={{ provider: "openrouter", model: "openai/gpt-4o-mini" }}
        onChange={vi.fn()}
      />,
    );

    expect(container).toBeEmptyDOMElement();
  });

  it("shows the current model on the trigger", () => {
    renderModelSelect(
      <ModelSelect
        options={[
          {
            provider: "openrouter",
            model: "openai/gpt-4o-mini",
            label: "GPT-4o mini",
            group: "Cloud",
          },
          {
            provider: "ollama",
            model: "gemma4:12b",
            label: "Gemma 4 12B",
            group: "Local",
          },
        ]}
        value={{ provider: "openrouter", model: "openai/gpt-4o-mini" }}
        onChange={vi.fn()}
      />,
    );

    expect(screen.getByRole("button", { name: /Model: GPT-4o mini/i })).toHaveTextContent(
      "GPT-4o mini",
    );
  });

  it("renders grouped options in dev", async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    const onChange = vi.fn();

    testRender(
      <TooltipProvider>
        <ModelSelect
          options={[
            {
              provider: "openrouter",
              model: "openai/gpt-4o-mini",
              label: "GPT-4o mini",
              group: "Cloud",
            },
            {
              provider: "ollama",
              model: "gemma4:12b",
              label: "Gemma 4 12B",
              group: "Local",
            },
          ]}
          value={{ provider: "openrouter", model: "openai/gpt-4o-mini" }}
          onChange={onChange}
        />
      </TooltipProvider>,
    );

    await user.click(screen.getByRole("button", { name: /Model: GPT-4o mini/i }));
    await user.click(
      await screen.findByRole("menuitem", { name: /Gemma 4 12B/i }),
    );

    await waitFor(() => {
      expect(onChange).toHaveBeenCalledWith({
        provider: "ollama",
        model: "gemma4:12b",
      });
    });
  });
});
