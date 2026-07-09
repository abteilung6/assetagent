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

const columnHelper = createColumnHelper<Transaction>();

const columns = [
  columnHelper.accessor("booking_date", {
    header: "Date",
  }),
  columnHelper.accessor("amount", {
    header: "Amount",
    cell: ({ getValue }) => (
      <span className="tabular-nums">{getValue()}</span>
    ),
  }),
  columnHelper.accessor("currency", {
    header: "Currency",
  }),
  columnHelper.accessor("counterparty", {
    header: "Counterparty",
  }),
  columnHelper.accessor("purpose", {
    header: "Purpose",
    cell: ({ getValue }) => (
      <span className="block max-w-xs truncate" title={getValue()}>
        {getValue()}
      </span>
    ),
  }),
  columnHelper.accessor("booking_text", {
    header: "Booking text",
    cell: ({ getValue }) => (
      <span className="block max-w-xs truncate" title={getValue()}>
        {getValue()}
      </span>
    ),
  }),
  columnHelper.accessor("order_account", {
    header: "Account",
  }),
];

type TransactionTableProps = {
  transactions: Transaction[];
};

export const TransactionTable: React.FC<TransactionTableProps> = ({
  transactions,
}) => {
  const table = useReactTable({
    data: transactions,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  return (
    <Table>
      <TableHeader>
        {table.getHeaderGroups().map((headerGroup) => (
          <TableRow key={headerGroup.id}>
            {headerGroup.headers.map((header) => (
              <TableHead key={header.id}>
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
          <TableRow key={row.id}>
            {row.getVisibleCells().map((cell) => (
              <TableCell key={cell.id}>
                {flexRender(cell.column.columnDef.cell, cell.getContext())}
              </TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
};
