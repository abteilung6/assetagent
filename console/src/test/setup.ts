import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, beforeEach, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import { mockApiResponse } from "@/test/mocks";

Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
});

window.scrollTo = vi.fn();

afterEach(() => {
  cleanup();
});

beforeEach(() => {
  vi.clearAllMocks();
  // Soft inbox badge queries run on every layout mount.
  vi.spyOn(sdk, "getTransferCandidates").mockResolvedValue(
    mockApiResponse({ data: [] }),
  );
  vi.spyOn(sdk, "getClassificationQueue").mockResolvedValue(
    mockApiResponse({ data: [] }),
  );
  vi.spyOn(sdk, "getCategories").mockResolvedValue(
    mockApiResponse({ data: [] }),
  );
  vi.spyOn(sdk, "getUncertainRecurring").mockResolvedValue(
    mockApiResponse({ data: [] }),
  );
  vi.spyOn(sdk, "getRecurring").mockResolvedValue(
    mockApiResponse({ data: [] }),
  );
});
