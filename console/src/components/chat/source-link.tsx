import { Link } from "@tanstack/react-router";
import { ArrowUpRight } from "lucide-react";
import type React from "react";

import type { TransactionSearchParams } from "@/pages/transactions/search-params";
import { cn } from "@/lib/utils";

import { buttonVariants } from "../ui/button";

type SourceLinkProps = {
  search: TransactionSearchParams;
  label?: string;
  className?: string;
};

export const SourceLink: React.FC<SourceLinkProps> = ({
  search,
  label = "View transactions",
  className,
}) => {
  return (
    <Link
      to="/transactions"
      search={search}
      className={cn(buttonVariants({ variant: "outline", size: "sm" }), className)}
    >
      {label}
      <ArrowUpRight className="size-3.5" />
    </Link>
  );
};
