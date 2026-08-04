import { Link } from "@tanstack/react-router";
import { MessageSquareIcon } from "lucide-react";
import type React from "react";

import type { ChatPageContext } from "@/api/types.gen";
import { buildChatHandoff } from "@/components/chat/starters";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type AskAboutThisProps = {
  prompt: string;
  context: ChatPageContext;
  className?: string;
};

export const AskAboutThis: React.FC<AskAboutThisProps> = ({
  prompt,
  context,
  className,
}) => {
  const handoff = buildChatHandoff({ prompt, context });
  return (
    <Link
      to={handoff.to}
      search={handoff.search}
      className={cn(
        buttonVariants({ variant: "ghost", size: "sm" }),
        "text-muted-foreground",
        className,
      )}
    >
      <MessageSquareIcon aria-hidden />
      Ask
    </Link>
  );
};
