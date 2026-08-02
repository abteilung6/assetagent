import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  getUncertainRecurringOptions,
  getUncertainRecurringQueryKey,
  postRecurringConfirmMutation,
  postRecurringRejectMutation,
} from "@/api/@tanstack/react-query.gen";
import type { RecurringSeries } from "@/api/types.gen";
import { apiErrorMessage } from "@/lib/api-error";

export function useUncertainRecurring() {
  return useQuery(getUncertainRecurringOptions());
}

export function useRecurringConfirm() {
  const queryClient = useQueryClient();
  return useMutation({
    ...postRecurringConfirmMutation(),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: getUncertainRecurringQueryKey(),
      });
    },
  });
}

export function useRecurringReject() {
  const queryClient = useQueryClient();
  return useMutation({
    ...postRecurringRejectMutation(),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: getUncertainRecurringQueryKey(),
      });
    },
  });
}

export function recurringActionErrorMessage(error: unknown): string {
  return apiErrorMessage(
    error,
    "Could not update this recurring series. Try again in a moment.",
  );
}

export type { RecurringSeries };
