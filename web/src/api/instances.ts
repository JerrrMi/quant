import { apiFetch } from "@/api/client";
import { API_PATHS } from "@/lib/api-paths";
import type {
  CreateStrategyInstanceBody,
  CreateStrategyInstanceResponseDTO,
  InstanceLifecycleActionDTO,
  StrategyInstanceDetailDTO,
  StrategyInstancesListDTO,
} from "@/types/strategies";

export async function fetchInstances(): Promise<StrategyInstancesListDTO> {
  return apiFetch<StrategyInstancesListDTO>(API_PATHS.console.instances);
}

export async function fetchInstanceDetail(
  id: string | number,
): Promise<StrategyInstanceDetailDTO> {
  return apiFetch<StrategyInstanceDetailDTO>(API_PATHS.console.instance(id));
}

export async function createInstance(
  body: CreateStrategyInstanceBody,
): Promise<CreateStrategyInstanceResponseDTO> {
  return apiFetch<CreateStrategyInstanceResponseDTO>(
    API_PATHS.console.instances,
    { method: "POST", json: body },
  );
}

export async function patchInstance(
  id: string | number,
  patch: Partial<{
    display_name: string;
    symbol: string;
    market_kind: "spot" | "futures";
    run_mode: "backtest" | "paper" | "live";
    agent_key: string;
    params: Record<string, unknown>;
  }>,
): Promise<{ status: string }> {
  return apiFetch<{ status: string }>(API_PATHS.console.instance(id), {
    method: "PATCH",
    json: patch,
  });
}

export async function instanceAction(
  id: string | number,
  action: InstanceLifecycleActionDTO,
): Promise<{ status: string }> {
  return apiFetch<{ status: string }>(API_PATHS.console.instanceActions(id), {
    method: "POST",
    json: { action },
  });
}
