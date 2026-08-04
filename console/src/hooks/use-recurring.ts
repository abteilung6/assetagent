import { useQuery } from "@tanstack/react-query";

import { getRecurringOptions } from "@/api/@tanstack/react-query.gen";

export function useRecurringSeries() {
  return useQuery(getRecurringOptions());
}
