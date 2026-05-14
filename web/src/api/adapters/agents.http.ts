import { apiFetch } from "@/api/client";
import { API_PATHS } from "@/lib/api-paths";
import type {
  AgentControlActionDTO,
  AgentControlResultDTO,
  AgentDetailDTO,
  AgentsListDTO,
} from "@/types/agents";

import type { AgentsApiAdapter } from "./agents.types";

async function fetchList(): Promise<AgentsListDTO> {
  return apiFetch<AgentsListDTO>(API_PATHS.agents.list, { method: "GET" });
}

async function fetchDetail(id: string): Promise<AgentDetailDTO> {
  return apiFetch<AgentDetailDTO>(API_PATHS.agents.detail(id), {
    method: "GET",
  });
}

async function postControl(
  id: string,
  action: AgentControlActionDTO,
): Promise<AgentControlResultDTO> {
  return apiFetch<AgentControlResultDTO>(API_PATHS.agents.control(id), {
    method: "POST",
    json: { action },
  });
}

export const agentsHttpAdapter: AgentsApiAdapter = {
  fetchList,
  fetchDetail,
  postControl,
};
