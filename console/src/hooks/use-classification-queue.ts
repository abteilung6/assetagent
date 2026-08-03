import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  getCategoriesOptions,
  getClassificationQueueOptions,
  getClassificationQueueQueryKey,
  postClassificationApplySuggestionsMutation,
  postClassificationCorrectMutation,
} from "@/api/@tanstack/react-query.gen";
import type {
  Category,
  ClassificationQueueItem,
} from "@/api/types.gen";
import { apiErrorMessage } from "@/lib/api-error";

export function useClassificationQueue() {
  return useQuery(getClassificationQueueOptions());
}

export function useCategories() {
  return useQuery(getCategoriesOptions());
}

export function useClassificationCorrect() {
  const queryClient = useQueryClient();
  return useMutation({
    ...postClassificationCorrectMutation(),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: getClassificationQueueQueryKey(),
      });
    },
  });
}

export function useClassificationApplySuggestions() {
  const queryClient = useQueryClient();
  return useMutation({
    ...postClassificationApplySuggestionsMutation(),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: getClassificationQueueQueryKey(),
      });
    },
  });
}

export function classificationActionErrorMessage(error: unknown): string {
  return apiErrorMessage(
    error,
    "Could not update this category. Try again in a moment.",
  );
}

export type { Category, ClassificationQueueItem };
