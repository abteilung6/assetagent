import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
  getBaselineMonthlyCashflowQueryKey,
  getClassificationQueueQueryKey,
  getCurrentBaselineQueryKey,
  getMoneyReviewsQueryKey,
  getTransactionsQueryKey,
  postTransactionOneOffMutation,
} from "@/api/@tanstack/react-query.gen";

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
          queryKey: getClassificationQueueQueryKey(),
        }),
        queryClient.invalidateQueries({ queryKey: getMoneyReviewsQueryKey() }),
      ]);
    },
  });
}
