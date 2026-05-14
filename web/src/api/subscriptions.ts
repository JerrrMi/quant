import { publicEnv } from "@/lib/env";

export type RealtimeHandlers = {
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (event: Event) => void;
  onMessage?: (data: unknown) => void;
};

/**
 * WebSocket subscription stub — connects when `NEXT_PUBLIC_WS_URL` is set.
 * Swap payload parsing when backend protocol stabilizes.
 */
export function subscribeWebSocket(handlers: RealtimeHandlers): () => void {
  const url = publicEnv.wsURL?.trim();
  if (!url || typeof WebSocket === "undefined") {
    return () => {};
  }

  const socket = new WebSocket(url);

  socket.addEventListener("open", () => handlers.onOpen?.());
  socket.addEventListener("close", () => handlers.onClose?.());
  socket.addEventListener("error", (event) => handlers.onError?.(event));
  socket.addEventListener("message", (event) => {
    try {
      const parsed = JSON.parse(event.data as string);
      handlers.onMessage?.(parsed);
    } catch {
      handlers.onMessage?.(event.data);
    }
  });

  return () => socket.close();
}

export type SSEHandlers = {
  onMessage?: (data: unknown) => void;
  onError?: (event: Event) => void;
};

/**
 * SSE stub — expects `NEXT_PUBLIC_SSE_BASE_URL` + relative stream path when wired up.
 */
export function subscribeSSE(
  relativePath: string,
  handlers: SSEHandlers,
): () => void {
  const base = publicEnv.sseBaseURL?.replace(/\/$/, "") ?? "";
  if (!base || typeof EventSource === "undefined") {
    return () => {};
  }

  const source = new EventSource(`${base}${relativePath}`, {
    withCredentials: true,
  });

  source.onmessage = (event) => {
    try {
      handlers.onMessage?.(JSON.parse(event.data));
    } catch {
      handlers.onMessage?.(event.data);
    }
  };

  source.onerror = (event) => handlers.onError?.(event);

  return () => source.close();
}
