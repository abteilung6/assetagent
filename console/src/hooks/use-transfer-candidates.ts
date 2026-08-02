import type React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  getTransferCandidatesOptions,
  getTransferCandidatesQueryKey,
  postTransferConfirmMutation,
  postTransferRejectMutation,
} from "@/api/@tanstack/react-query.gen";
import type { TransferCandidate } from "@/api/types.gen";
import { apiErrorMessage } from "@/lib/api-error";

export function useTransferCandidates() {
  return useQuery(getTransferCandidatesOptions());
}

export function useTransferConfirm() {
  const queryClient = useQueryClient();
  return useMutation({
    ...postTransferConfirmMutation(),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: getTransferCandidatesQueryKey(),
      });
    },
  });
}

export function useTransferReject() {
  const queryClient = useQueryClient();
  return useMutation({
    ...postTransferRejectMutation(),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: getTransferCandidatesQueryKey(),
      });
    },
  });
}

export function transferActionErrorMessage(error: unknown): string {
  return apiErrorMessage(
    error,
    "Could not update this transfer. Try again in a moment.",
  );
}

export type { TransferCandidate };
