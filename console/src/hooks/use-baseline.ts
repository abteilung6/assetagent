import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  getCurrentBaselineOptions,
  getCurrentBaselineQueryKey,
  postBaselineAdjustMutation,
  postBaselineConfirmMutation,
  postBaselinesRecomputeMutation,
} from "@/api/@tanstack/react-query.gen";
import type { FinancialBaseline } from "@/api/types.gen";
import { apiErrorMessage } from "@/lib/api-error";

export function useCurrentBaseline() {
  return useQuery({
    ...getCurrentBaselineOptions(),
    retry: false,
  });
}

export function useBaselineRecompute() {
  const queryClient = useQueryClient();
  return useMutation({
    ...postBaselinesRecomputeMutation(),
    onSuccess: async (data) => {
      queryClient.setQueryData(getCurrentBaselineQueryKey(), data);
      await queryClient.invalidateQueries({
        queryKey: getCurrentBaselineQueryKey(),
      });
    },
  });
}

export function useBaselineConfirm() {
  const queryClient = useQueryClient();
  return useMutation({
    ...postBaselineConfirmMutation(),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: getCurrentBaselineQueryKey(),
      });
    },
  });
}

export function useBaselineAdjust() {
  const queryClient = useQueryClient();
  return useMutation({
    ...postBaselineAdjustMutation(),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: getCurrentBaselineQueryKey(),
      });
    },
  });
}

export function isBaselineMissing(error: unknown): boolean {
  if (typeof error !== "object" || error === null) {
    return false;
  }
  const record = error as {
    response?: { status?: number };
    status?: number;
    error?: string;
    message?: string;
  };
  if (record.response?.status === 404 || record.status === 404) {
    return true;
  }
  if (record.error === "not_found") {
    return true;
  }
  return (
    typeof record.message === "string" &&
    /no baseline/i.test(record.message)
  );
}

export function baselineActionErrorMessage(error: unknown): string {
  return apiErrorMessage(
    error,
    "Could not update the baseline. Try again in a moment.",
  );
}

export type { FinancialBaseline };
