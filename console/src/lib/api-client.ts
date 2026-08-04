import { client } from "@/api/client.gen";

const baseUrl = import.meta.env.VITE_API_BASE_URL || "";

client.setConfig({
  baseUrl,
  credentials: "include",
});

export function apiBaseUrl(): string {
  return baseUrl;
}
