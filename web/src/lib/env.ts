export const publicEnv = {
  appName: process.env.NEXT_PUBLIC_APP_NAME ?? "AltShort Console",
  apiBaseURL: process.env.NEXT_PUBLIC_API_BASE_URL ?? "",
  wsURL: process.env.NEXT_PUBLIC_WS_URL ?? "",
  sseBaseURL: process.env.NEXT_PUBLIC_SSE_BASE_URL ?? "",
  deployEnv: process.env.NEXT_PUBLIC_DEPLOY_ENV ?? "development",
  devMockAuth: process.env.NEXT_PUBLIC_DEV_MOCK_AUTH === "true",
} as const;
