import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from "@tanstack/react-table";
import type React from "react";

import type { Transaction } from "@/api/types.gen";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";

const columnHelper = createColumnHelper<Transaction>();

function formatAmount(amount: string, currency: string) {
  return `${amount} ${currency}`;
}

const columns = [
  columnHelper.accessor("booking_date", {
    header: "Date",
  }),
  columnHelper.accessor("counterparty", {
    header: "Counterparty",
    cell: ({ row }) => {
      const value = row.original.counterparty;
      return (
        <span className="flex min-w-0 items-center gap-1.5">
          <span className="block truncate" title={value}>
            {value}
          </span>
          {row.original.one_off ? (
            <span className="shrink-0 text-[10px] font-medium tracking-wide text-muted-foreground uppercase">
              One-off
            </span>
          ) : null}
        </span>
      );
    },
  }),
  columnHelper.accessor("purpose", {
    header: "Purpose",
    cell: ({ getValue }) => {
      const value = getValue();
      return (
        <span className="block truncate" title={value}>
          {value}
        </span>
      );
    },
  }),
  columnHelper.accessor("amount", {
    header: () => <span className="block w-full text-right">Amount</span>,
    cell: ({ row }) => (
      <span className="block text-right tabular-nums whitespace-nowrap">
        {formatAmount(row.original.amount, row.original.currency)}
      </span>
    ),
  }),
];

function columnHeadClassName(columnId: string) {
  switch (columnId) {
    case "booking_date":
      return "w-[6.5rem]";
    case "counterparty":
      return "w-[34%]";
    case "purpose":
      return "w-auto";
    case "amount":
      return "w-[7.5rem]";
    default:
      return undefined;
  }
}

function columnCellClassName(columnId: string) {
  return cn(
    columnId === "amount" && "text-right",
    (columnId === "counterparty" || columnId === "purpose") && "max-w-0",
  );
}

type TransactionTableProps = {
  transactions: Transaction[];
  onRowClick?: (transaction: Transaction) => void;
};

export const TransactionTable: React.FC<TransactionTableProps> = ({
  transactions,
  onRowClick,
}) => {
  const table = useReactTable({
    data: transactions,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  return (
    <Table className="table-fixed">
      <colgroup>
        <col style={{ width: "6.5rem" }} />
        <col style={{ width: "34%" }} />
        <col />
        <col style={{ width: "7.5rem" }} />
      </colgroup>
      <TableHeader>
        {table.getHeaderGroups().map((headerGroup) => (
          <TableRow key={headerGroup.id}>
            {headerGroup.headers.map((header) => (
              <TableHead
                key={header.id}
                className={columnHeadClassName(header.column.id)}
              >
                {header.isPlaceholder
                  ? null
                  : flexRender(
                      header.column.columnDef.header,
                      header.getContext(),
                    )}
              </TableHead>
            ))}
          </TableRow>
        ))}
      </TableHeader>
      <TableBody>
        {table.getRowModel().rows.map((row) => (
          <TableRow
            key={row.id}
            className={onRowClick ? "cursor-pointer" : undefined}
            onClick={() => onRowClick?.(row.original)}
          >
            {row.getVisibleCells().map((cell) => (
              <TableCell
                key={cell.id}
                className={columnCellClassName(cell.column.id)}
              >
                {flexRender(cell.column.columnDef.cell, cell.getContext())}
              </TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
};
