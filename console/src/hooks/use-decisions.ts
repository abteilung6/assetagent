import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  getActionsOptions,
  getActionsQueryKey,
  getDecisionsQueryKey,
  postActionStatusMutation,
  postDecisionsMutation,
} from "@/api/@tanstack/react-query.gen";
import type { Action, Decision } from "@/api/types.gen";
import { apiErrorMessage } from "@/lib/api-error";

export function useOpenActions(limit = 50) {
  return useQuery(
    getActionsOptions({ query: { status: "planned", limit } }),
  );
}

export function useCreateDecision() {
  const queryClient = useQueryClient();
  return useMutation({
    ...postDecisionsMutation(),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: getDecisionsQueryKey() });
      await queryClient.invalidateQueries({ queryKey: getActionsQueryKey() });
    },
  });
}

export function useUpdateActionStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    ...postActionStatusMutation(),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: getActionsQueryKey() });
      await queryClient.invalidateQueries({ queryKey: getDecisionsQueryKey() });
    },
  });
}

export function decisionActionErrorMessage(error: unknown): string {
  return apiErrorMessage(
    error,
    "Could not update this action. Try again in a moment.",
  );
}

export function dueInDays(days: number): string {
  const d = new Date();
  d.setUTCDate(d.getUTCDate() + days);
  return d.toISOString().slice(0, 10);
}

export type { Action, Decision };
