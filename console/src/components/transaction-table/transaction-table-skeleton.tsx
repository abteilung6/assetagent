import type React from "react";

import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const SKELETON_ROWS = 5;
const SKELETON_COLUMNS = 4;

export const TransactionTableSkeleton: React.FC = () => {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          {Array.from({ length: SKELETON_COLUMNS }, (_, index) => (
            <TableHead key={index}>
              <Skeleton className="h-4 w-20" />
            </TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {Array.from({ length: SKELETON_ROWS }, (_, rowIndex) => (
          <TableRow key={rowIndex}>
            {Array.from({ length: SKELETON_COLUMNS }, (_, colIndex) => (
              <TableCell key={colIndex}>
                <Skeleton className="h-4 w-full max-w-32" />
              </TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
};
