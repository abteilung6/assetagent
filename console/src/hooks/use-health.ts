import { useQuery } from "@tanstack/react-query";

import { getHealthOptions } from "@/api/@tanstack/react-query.gen";

export function useHealth() {
  return useQuery({
    ...getHealthOptions(),
    retry: false,
    refetchInterval: 30_000,
  });
}
