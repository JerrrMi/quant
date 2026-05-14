import { publicEnv } from "@/lib/env";

import { agentsHttpAdapter } from "./adapters/agents.http";
import { agentsMockAdapter } from "./adapters/agents.mock";
import type {
  AgentControlActionDTO,
  AgentControlResultDTO,
  AgentDetailDTO,
  AgentsListDTO,
} from "@/types/agents";

const adapter = publicEnv.useMockApi ? agentsMockAdapter : agentsHttpAdapter;

export async function fetchAgentsList(): Promise<AgentsListDTO> {
  return adapter.fetchList();
}

export async function fetchAgentDetail(id: string): Promise<AgentDetailDTO> {
  return adapter.fetchDetail(id);
}

export async function postAgentControl(
  id: string,
  action: AgentControlActionDTO,
): Promise<AgentControlResultDTO> {
  return adapter.postControl(id, action);
}
