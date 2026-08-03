import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  getForecastScenariosOptions,
  getForecastScenariosQueryKey,
  getLatestForecastOptions,
  getLatestForecastQueryKey,
  postForecastScenarioMutation,
  postForecastsMutation,
} from "@/api/@tanstack/react-query.gen";
import type { Forecast, Scenario } from "@/api/types.gen";
import { apiErrorMessage } from "@/lib/api-error";

export function useLatestForecast() {
  return useQuery({
    ...getLatestForecastOptions(),
    retry: false,
  });
}

export function useCreateForecast() {
  const queryClient = useQueryClient();
  return useMutation({
    ...postForecastsMutation(),
    onSuccess: async (data) => {
      queryClient.setQueryData(getLatestForecastQueryKey(), data);
      await queryClient.invalidateQueries({
        queryKey: getLatestForecastQueryKey(),
      });
    },
  });
}

export function useForecastScenarios(forecastId: string | undefined) {
  return useQuery({
    ...getForecastScenariosOptions({ path: { id: forecastId! } }),
    enabled: Boolean(forecastId),
  });
}

export function useRunScenario(forecastId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    ...postForecastScenarioMutation(),
    onSuccess: async () => {
      if (!forecastId) {
        return;
      }
      await queryClient.invalidateQueries({
        queryKey: getForecastScenariosQueryKey({ path: { id: forecastId } }),
      });
    },
  });
}

export function isForecastMissing(error: unknown): boolean {
  if (typeof error !== "object" || error === null) {
    return false;
  }
  const record = error as { error?: string; message?: string };
  return (
    record.error === "not_found" ||
    (typeof record.message === "string" && /no forecast/i.test(record.message))
  );
}

export function forecastActionErrorMessage(error: unknown): string {
  return apiErrorMessage(
    error,
    "Could not update the forecast. Try again in a moment.",
  );
}

export type { Forecast, Scenario };
