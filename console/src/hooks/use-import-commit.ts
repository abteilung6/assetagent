import { useMutation } from "@tanstack/react-query";

import { postImportsMutation } from "@/api/@tanstack/react-query.gen";
import type { ImportCommitResponse } from "@/api/types.gen";
import { apiErrorMessage } from "@/lib/api-error";

export function useImportCommit() {
  return useMutation(postImportsMutation());
}

export function commitErrorMessage(error: unknown): string {
  return apiErrorMessage(
    error,
    "Could not import this file. Try again or choose another export.",
  );
}

export type { ImportCommitResponse };
