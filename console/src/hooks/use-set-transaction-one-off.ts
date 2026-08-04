import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
  getBaselineMonthlyCashflowQueryKey,
  getClassificationQueueQueryKey,
  getCurrentBaselineQueryKey,
  getMoneyReviewsQueryKey,
  getTransactionsQueryKey,
  postTransactionOneOffMutation,
} from "@/api/@tanstack/react-query.gen";

function isBaselineInsightQuery(queryKey: readonly unknown[]): boolean {
  const key = queryKey[0];
  return (
    typeof key === "object" &&
    key !== null &&
    "_id" in key &&
    (key._id === "getBaselineOneOffImpact" ||
      key._id === "getBaselineCategorySpend")
  );
}

export function useSetTransactionOneOff() {
  const queryClient = useQueryClient();

  return useMutation({
    ...postTransactionOneOffMutation(),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: getTransactionsQueryKey() }),
        queryClient.invalidateQueries({
          queryKey: getCurrentBaselineQueryKey(),
        }),
        queryClient.invalidateQueries({
          queryKey: getBaselineMonthlyCashflowQueryKey(),
        }),
        queryClient.invalidateQueries({
          predicate: (query) => isBaselineInsightQuery(query.queryKey),
        }),
        queryClient.invalidateQueries({
          queryKey: getClassificationQueueQueryKey(),
        }),
        queryClient.invalidateQueries({ queryKey: getMoneyReviewsQueryKey() }),
      ]);
    },
  });
}
