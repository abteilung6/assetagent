import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  getImportsOptions,
  getImportsQueryKey,
  getTransactionsQueryKey,
  postImportRollbackMutation,
} from "@/api/@tanstack/react-query.gen";
import type { ImportRun } from "@/api/types.gen";
import { apiErrorMessage } from "@/lib/api-error";

const importRunsQuery = { query: { limit: 20 } } as const;

export function useImportRuns() {
  return useQuery(getImportsOptions(importRunsQuery));
}

export function useImportRollback() {
  const queryClient = useQueryClient();
  return useMutation({
    ...postImportRollbackMutation(),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: getImportsQueryKey(importRunsQuery),
        }),
        queryClient.invalidateQueries({
          queryKey: getTransactionsQueryKey(),
        }),
      ]);
    },
  });
}

export function invalidateImportRuns(queryClient: ReturnType<typeof useQueryClient>) {
  return queryClient.invalidateQueries({
    queryKey: getImportsQueryKey(importRunsQuery),
  });
}

export function rollbackErrorMessage(error: unknown): string {
  return apiErrorMessage(
    error,
    "Could not undo this import. Try again in a moment.",
  );
}

export type { ImportRun };
