export function mockApiResponse<TData>(data: TData) {
  return {
    data,
    request: new Request("http://localhost/api/test"),
    response: new Response(JSON.stringify(data), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  };
}
