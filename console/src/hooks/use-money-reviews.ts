import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  getMoneyReviewOptions,
  getMoneyReviewsOptions,
  getMoneyReviewsQueryKey,
  getMoneyReviewQueryKey,
  postMoneyReviewConfirmMutation,
  postMoneyReviewsMutation,
} from "@/api/@tanstack/react-query.gen";
import type { MoneyReview } from "@/api/types.gen";
import { apiErrorMessage } from "@/lib/api-error";

export function useMoneyReviews(limit = 50) {
  return useQuery(getMoneyReviewsOptions({ query: { limit } }));
}

export function useMoneyReview(id: string | undefined) {
  return useQuery({
    ...getMoneyReviewOptions({ path: { id: id! } }),
    enabled: Boolean(id),
  });
}

export function useCreateMoneyReview() {
  const queryClient = useQueryClient();
  return useMutation({
    ...postMoneyReviewsMutation(),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: getMoneyReviewsQueryKey(),
      });
    },
  });
}

export function useConfirmMoneyReview() {
  const queryClient = useQueryClient();
  return useMutation({
    ...postMoneyReviewConfirmMutation(),
    onSuccess: async (_data, vars) => {
      await queryClient.invalidateQueries({
        queryKey: getMoneyReviewsQueryKey(),
      });
      if (vars.path?.id) {
        await queryClient.invalidateQueries({
          queryKey: getMoneyReviewQueryKey({ path: { id: vars.path.id } }),
        });
      }
    },
  });
}

export function moneyReviewActionErrorMessage(error: unknown): string {
  return apiErrorMessage(
    error,
    "Could not update this money review. Try again in a moment.",
  );
}

export type { MoneyReview };
