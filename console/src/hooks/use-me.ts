import { useQuery } from "@tanstack/react-query";

import { getMeOptions } from "@/api/@tanstack/react-query.gen";

export function useMe() {
  return useQuery({
    ...getMeOptions(),
    retry: false,
  });
}
