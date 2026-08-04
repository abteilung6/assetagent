import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  getBaselineCategoryMerchantsOptions,
  getBaselineCategorySpendMonthlyOptions,
  getBaselineCategorySpendOptions,
  getBaselineDailyExpensePaceOptions,
  getBaselineMonthlyCashflowOptions,
  getBaselineOneOffImpactOptions,
  getCurrentBaselineOptions,
  getCurrentBaselineQueryKey,
  getBaselineMonthlyCashflowQueryKey,
  postBaselineAdjustMutation,
  postBaselineConfirmMutation,
  postBaselinesRecomputeMutation,
} from "@/api/@tanstack/react-query.gen";
import type {
  BaselineMonthlyCashflowPoint,
  FinancialBaseline,
} from "@/api/types.gen";
import { apiErrorMessage } from "@/lib/api-error";

export function useCurrentBaseline() {
  return useQuery({
    ...getCurrentBaselineOptions(),
    retry: false,
  });
}

export function useBaselineMonthlyCashflow(months = 6) {
  return useQuery({
    ...getBaselineMonthlyCashflowOptions({ query: { months } }),
    retry: false,
  });
}

export function useBaselineOneOffImpact(from: string, to: string, enabled = true) {
  return useQuery({
    ...getBaselineOneOffImpactOptions({
      query: { from, to },
    }),
    enabled: enabled && Boolean(from) && Boolean(to),
    retry: false,
  });
}

export function useBaselineCategorySpend(
  from: string,
  to: string,
  limit = 8,
  enabled = true,
) {
  return useQuery({
    ...getBaselineCategorySpendOptions({
      query: { from, to, limit },
    }),
    enabled: enabled && Boolean(from) && Boolean(to),
    retry: false,
  });
}

export function useBaselineCategoryMerchants(
  from: string,
  to: string,
  categorySlug: string,
  limit = 8,
  enabled = true,
) {
  return useQuery({
    ...getBaselineCategoryMerchantsOptions({
      query: { from, to, category_slug: categorySlug, limit },
    }),
    enabled:
      enabled && Boolean(from) && Boolean(to) && Boolean(categorySlug),
    retry: false,
  });
}

export function useBaselineCategorySpendMonthly(
  from: string,
  to: string,
  limit = 5,
  enabled = true,
) {
  return useQuery({
    ...getBaselineCategorySpendMonthlyOptions({
      query: { from, to, limit },
    }),
    enabled: enabled && Boolean(from) && Boolean(to),
    retry: false,
  });
}

export function useBaselineDailyExpensePace(
  from: string,
  to: string,
  enabled = true,
) {
  return useQuery({
    ...getBaselineDailyExpensePaceOptions({
      query: { from, to },
    }),
    enabled: enabled && Boolean(from) && Boolean(to),
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
      await queryClient.invalidateQueries({
        queryKey: getBaselineMonthlyCashflowQueryKey(),
      });
      await queryClient.invalidateQueries({
        predicate: (query) => {
          const key = query.queryKey[0];
          return (
            typeof key === "object" &&
            key !== null &&
            "_id" in key &&
            (key._id === "getBaselineOneOffImpact" ||
              key._id === "getBaselineCategorySpend" ||
              key._id === "getBaselineCategoryMerchants" ||
              key._id === "getBaselineCategorySpendMonthly" ||
              key._id === "getBaselineDailyExpensePace")
          );
        },
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

export type { BaselineMonthlyCashflowPoint, FinancialBaseline };
