import { CheckIcon, ChevronDownIcon } from "lucide-react";
import { useEffect, useState } from "react";

import type { LlmModelOption, LlmModelSelection } from "@/api/types.gen";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

type ModelSelectProps = {
  options: LlmModelOption[];
  value: LlmModelSelection;
  onChange: (selection: LlmModelSelection) => void;
  disabled?: boolean;
  className?: string;
};

function isSelected(option: LlmModelOption, value: LlmModelSelection): boolean {
  return option.provider === value.provider && option.model === value.model;
}

function selectedLabel(
  options: LlmModelOption[],
  value: LlmModelSelection,
): string {
  return (
    options.find((option) => isSelected(option, value))?.label ?? value.model
  );
}

function useMenuPlacement() {
  const [placement, setPlacement] = useState<{
    side: "top" | "bottom";
    align: "start" | "end";
  }>({ side: "top", align: "start" });

  useEffect(() => {
    const media = window.matchMedia("(min-width: 640px)");
    const update = () => {
      setPlacement(
        media.matches
          ? { side: "bottom", align: "end" }
          : { side: "top", align: "start" },
      );
    };
    update();
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, []);

  return placement;
}

export function ModelSelect({
  options,
  value,
  onChange,
  disabled = false,
  className,
}: ModelSelectProps) {
  if (options.length <= 1) {
    return null;
  }

  const groups = new Map<string, LlmModelOption[]>();
  const ungrouped: LlmModelOption[] = [];

  for (const option of options) {
    if (option.group) {
      const existing = groups.get(option.group) ?? [];
      existing.push(option);
      groups.set(option.group, existing);
    } else {
      ungrouped.push(option);
    }
  }

  const label = selectedLabel(options, value);
  const { side, align } = useMenuPlacement();

  const renderItem = (option: LlmModelOption) => (
    <DropdownMenuItem
      key={`${option.provider}:${option.model}`}
      onClick={() =>
        onChange({ provider: option.provider, model: option.model })
      }
    >
      <span className="flex-1 truncate">{option.label}</span>
      {isSelected(option, value) ? (
        <CheckIcon className="size-3.5 text-muted-foreground" />
      ) : null}
    </DropdownMenuItem>
  );

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={disabled}
            aria-label={`Model: ${label}`}
            className={cn(
              "h-7 max-w-full gap-0.5 rounded-full border-border/60 bg-background/60 px-2 text-[11px] font-normal text-muted-foreground shadow-none hover:bg-muted/60 sm:h-8 sm:gap-1 sm:px-2.5 sm:text-xs",
              className,
            )}
          />
        }
      >
        <span className="min-w-0 truncate sm:max-w-[7rem] md:max-w-[9rem]">
          {label}
        </span>
        <ChevronDownIcon className="size-3 shrink-0 opacity-60 sm:size-3.5" />
      </DropdownMenuTrigger>
      <DropdownMenuContent
        side={side}
        align={align}
        className="max-h-64 min-w-40 max-w-[min(16rem,calc(100vw-2rem))] overflow-y-auto"
      >
        {ungrouped.map(renderItem)}
        {[...groups.entries()].map(([group, groupOptions]) => (
          <DropdownMenuGroup key={group}>
            <DropdownMenuLabel>{group}</DropdownMenuLabel>
            {groupOptions.map(renderItem)}
          </DropdownMenuGroup>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
